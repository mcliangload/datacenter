package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"datacenter/internal/model"
)

// ItemStore 数据项数据访问
type ItemStore struct {
	coll *mongo.Collection
}

// NewItemStore 构造数据项存储
func NewItemStore(db *mongo.Database) *ItemStore {
	return &ItemStore{coll: db.Collection(model.CollectionDataItems)}
}

// Create 创建数据项；同集合内路径重复时返回重复键错误
func (s *ItemStore) Create(ctx context.Context, item *model.DataItem) error {
	item.ID = primitive.NewObjectID()
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = now
	}
	_, err := s.coll.InsertOne(ctx, item)
	return err
}

// InsertMany 批量插入数据项（种子数据/批量导入）。
// 未设置 ID/时间戳的项自动补齐；重复键（同集合内路径重复）返回批量插入错误。
func (s *ItemStore) InsertMany(ctx context.Context, items []*model.DataItem) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now()
	docs := make([]interface{}, 0, len(items))
	for _, it := range items {
		if it.ID.IsZero() {
			it.ID = primitive.NewObjectID()
		}
		if it.CreatedAt.IsZero() {
			it.CreatedAt = now
		}
		if it.UpdatedAt.IsZero() {
			it.UpdatedAt = now
		}
		docs = append(docs, it)
	}
	_, err := s.coll.InsertMany(ctx, docs)
	return err
}

// FindByID 按 ID 查询，不存在时返回 (nil, nil)
func (s *ItemStore) FindByID(ctx context.Context, id primitive.ObjectID) (*model.DataItem, error) {
	var it model.DataItem
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&it)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// FindByIDs 批量查询数据项（结果无序，缺失的 id 不返回）
func (s *ItemStore) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*model.DataItem, error) {
	if len(ids) == 0 {
		return []*model.DataItem{}, nil
	}
	cursor, err := s.coll.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]*model.DataItem, 0, len(ids))
	for cursor.Next(ctx) {
		var it model.DataItem
		if err := cursor.Decode(&it); err != nil {
			return nil, err
		}
		items = append(items, &it)
	}
	return items, cursor.Err()
}

// FindByAncestor 按物化路径查询子树（ancestors 包含某节点）
func (s *ItemStore) FindByAncestor(ctx context.Context, ancestorID primitive.ObjectID) ([]*model.DataItem, error) {
	cursor, err := s.coll.Find(ctx, bson.M{"ancestors": ancestorID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*model.DataItem
	for cursor.Next(ctx) {
		var it model.DataItem
		if err := cursor.Decode(&it); err != nil {
			return nil, err
		}
		items = append(items, &it)
	}
	return items, cursor.Err()
}

// List 分页查询数据项，按创建时间倒序
func (s *ItemStore) List(ctx context.Context, filter bson.M, page, pageSize int64) ([]*model.DataItem, int64, error) {
	return s.ListWithSort(ctx, filter, bson.D{{Key: "created_at", Value: -1}}, page, pageSize)
}

// ListWithSort 分页查询数据项（自定义排序；系统优化 1.2：DQL ORDER BY）
func (s *ItemStore) ListWithSort(ctx context.Context, filter bson.M, sort bson.D, page, pageSize int64) ([]*model.DataItem, int64, error) {
	total, err := s.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(sort)
	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	items := make([]*model.DataItem, 0, pageSize)
	for cursor.Next(ctx) {
		var it model.DataItem
		if err := cursor.Decode(&it); err != nil {
			return nil, 0, err
		}
		items = append(items, &it)
	}
	return items, total, cursor.Err()
}

// UpdateFields 按 ID 更新字段（自动刷新 updated_at）
func (s *ItemStore) UpdateFields(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
	fields["updated_at"] = time.Now()
	res, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": fields})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Delete 删除数据项（仅元数据）
func (s *ItemStore) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := s.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByIDs 批量删除数据项（级联删除用）
func (s *ItemStore) DeleteByIDs(ctx context.Context, ids []primitive.ObjectID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.coll.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}

// DeleteByCollection 级联删除集合下全部数据项
func (s *ItemStore) DeleteByCollection(ctx context.Context, collectionID primitive.ObjectID) error {
	_, err := s.coll.DeleteMany(ctx, bson.M{"collection_id": collectionID})
	return err
}

// Count 统计数据项数（支持任意过滤条件）
func (s *ItemStore) Count(ctx context.Context, filter bson.M) (int64, error) {
	return s.coll.CountDocuments(ctx, filter)
}

// Aggregate 聚合管道（系统优化 1.2：DQL 分组统计等）
func (s *ItemStore) Aggregate(ctx context.Context, pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	return s.coll.Aggregate(ctx, pipeline)
}

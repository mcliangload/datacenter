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

// RelationStore 数据项关联边数据访问
type RelationStore struct {
	coll *mongo.Collection
}

// NewRelationStore 构造关联边存储
func NewRelationStore(db *mongo.Database) *RelationStore {
	return &RelationStore{coll: db.Collection(model.CollectionRelations)}
}

// Create 创建边；重复键（含单父冲突）返回重复键错误
func (s *RelationStore) Create(ctx context.Context, r *model.Relation) error {
	r.ID = primitive.NewObjectID()
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = now
	}
	_, err := s.coll.InsertOne(ctx, r)
	return err
}

// InsertMany 批量插入边（种子数据/批量导入）。
// 未设置 ID/时间戳的边自动补齐；重复键（from/to/type 或单父冲突）返回批量插入错误。
func (s *RelationStore) InsertMany(ctx context.Context, rels []*model.Relation) error {
	if len(rels) == 0 {
		return nil
	}
	now := time.Now()
	docs := make([]interface{}, 0, len(rels))
	for _, r := range rels {
		if r.ID.IsZero() {
			r.ID = primitive.NewObjectID()
		}
		if r.CreatedAt.IsZero() {
			r.CreatedAt = now
		}
		if r.UpdatedAt.IsZero() {
			r.UpdatedAt = now
		}
		docs = append(docs, r)
	}
	_, err := s.coll.InsertMany(ctx, docs)
	return err
}

// FindByID 按 ID 查询，不存在返回 (nil, nil)
func (s *RelationStore) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Relation, error) {
	var r model.Relation
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&r)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// List 分页查询边（按创建时间倒序），支持任意过滤条件
func (s *RelationStore) List(ctx context.Context, filter bson.M, page, pageSize int64) ([]*model.Relation, int64, error) {
	total, err := s.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	rels := make([]*model.Relation, 0, pageSize)
	for cursor.Next(ctx) {
		var r model.Relation
		if err := cursor.Decode(&r); err != nil {
			return nil, 0, err
		}
		rels = append(rels, &r)
	}
	return rels, total, cursor.Err()
}

// Count 统计边数（支持任意过滤条件）
func (s *RelationStore) Count(ctx context.Context, filter bson.M) (int64, error) {
	return s.coll.CountDocuments(ctx, filter)
}

// UpdateMeta 更新边属性
func (s *RelationStore) UpdateMeta(ctx context.Context, id primitive.ObjectID, meta map[string]interface{}) error {
	res, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"meta": meta, "updated_at": time.Now()}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByID 删除单条边
func (s *RelationStore) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	res, err := s.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByCollection 删除起点属于某集合的全部边（集合级联）
func (s *RelationStore) DeleteByCollection(ctx context.Context, collectionID primitive.ObjectID) error {
	_, err := s.coll.DeleteMany(ctx, bson.M{"collection_id": collectionID})
	return err
}

// DeleteByItems 删除与给定数据项集合相关的全部边（出边或入边）
func (s *RelationStore) DeleteByItems(ctx context.Context, itemIDs []primitive.ObjectID) error {
	if len(itemIDs) == 0 {
		return nil
	}
	_, err := s.coll.DeleteMany(ctx, bson.M{"$or": []bson.M{
		{"from_item_id": bson.M{"$in": itemIDs}},
		{"to_item_id": bson.M{"$in": itemIDs}},
	}})
	return err
}

// ChildrenOf 查询某数据项的全部父子出边（子节点）
func (s *RelationStore) ChildrenOf(ctx context.Context, parentID primitive.ObjectID) ([]*model.Relation, error) {
	return s.listAll(ctx, bson.M{"type": model.RelationParentChild, "from_item_id": parentID})
}

// ParentEdge 查询某数据项的父子入边（父节点），无则返回 (nil, nil)
func (s *RelationStore) ParentEdge(ctx context.Context, childID primitive.ObjectID) (*model.Relation, error) {
	var r model.Relation
	err := s.coll.FindOne(ctx, bson.M{"type": model.RelationParentChild, "to_item_id": childID}).Decode(&r)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AncestorsOf 收集某数据项的祖先链（沿父子入边上溯，深度上限内）
func (s *RelationStore) AncestorsOf(ctx context.Context, itemID primitive.ObjectID, maxDepth int) ([]primitive.ObjectID, error) {
	var chain []primitive.ObjectID
	cur := itemID
	for i := 0; i < maxDepth; i++ {
		edge, err := s.ParentEdge(ctx, cur)
		if err != nil {
			return nil, err
		}
		if edge == nil {
			break
		}
		chain = append(chain, edge.FromItemID)
		cur = edge.FromItemID
	}
	return chain, nil
}

// DescendantIDs 沿父子出边 BFS 收集子树（含自身），深度上限内
func (s *RelationStore) DescendantIDs(ctx context.Context, rootID primitive.ObjectID, maxDepth int) ([]primitive.ObjectID, error) {
	ids := []primitive.ObjectID{rootID}
	level := []primitive.ObjectID{rootID}
	for i := 0; i < maxDepth && len(level) > 0; i++ {
		edges, err := s.listAll(ctx, bson.M{"type": model.RelationParentChild, "from_item_id": bson.M{"$in": level}})
		if err != nil {
			return nil, err
		}
		next := make([]primitive.ObjectID, 0, len(edges))
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			seen[id.Hex()] = true
		}
		for _, e := range edges {
			if !seen[e.ToItemID.Hex()] {
				seen[e.ToItemID.Hex()] = true
				ids = append(ids, e.ToItemID)
				next = append(next, e.ToItemID)
			}
		}
		level = next
	}
	return ids, nil
}

// Aggregate 聚合管道（徽标统计等）
func (s *RelationStore) Aggregate(ctx context.Context, pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	return s.coll.Aggregate(ctx, pipeline)
}

// listAll 无条件分页地取全部匹配边（内部用，子节点/祖先查询）
func (s *RelationStore) listAll(ctx context.Context, filter bson.M) ([]*model.Relation, error) {
	cursor, err := s.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rels []*model.Relation
	for cursor.Next(ctx) {
		var r model.Relation
		if err := cursor.Decode(&r); err != nil {
			return nil, err
		}
		rels = append(rels, &r)
	}
	return rels, cursor.Err()
}

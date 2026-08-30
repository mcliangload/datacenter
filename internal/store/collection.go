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

// CollectionStore 业务集合数据访问
type CollectionStore struct {
	coll *mongo.Collection
}

// NewCollectionStore 构造业务集合存储
func NewCollectionStore(db *mongo.Database) *CollectionStore {
	return &CollectionStore{coll: db.Collection(model.CollectionBusiness)}
}

// Create 创建集合；名称重复时返回重复键错误
func (s *CollectionStore) Create(ctx context.Context, c *model.BusinessCollection) error {
	c.ID = primitive.NewObjectID()
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	_, err := s.coll.InsertOne(ctx, c)
	return err
}

// FindByID 按 ID 查询，不存在时返回 (nil, nil)
func (s *CollectionStore) FindByID(ctx context.Context, id primitive.ObjectID) (*model.BusinessCollection, error) {
	var c model.BusinessCollection
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// List 分页查询集合，按创建时间倒序
func (s *CollectionStore) List(ctx context.Context, filter bson.M, page, pageSize int64) ([]*model.BusinessCollection, int64, error) {
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

	cols := make([]*model.BusinessCollection, 0, pageSize)
	for cursor.Next(ctx) {
		var c model.BusinessCollection
		if err := cursor.Decode(&c); err != nil {
			return nil, 0, err
		}
		cols = append(cols, &c)
	}
	return cols, total, cursor.Err()
}

// UpdateMeta 修改集合基础信息
func (s *CollectionStore) UpdateMeta(ctx context.Context, id primitive.ObjectID, description string) error {
	_, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"description": description, "updated_at": time.Now()}})
	return err
}

// UpdateTagSchema 全量替换标签定义
func (s *CollectionStore) UpdateTagSchema(ctx context.Context, id primitive.ObjectID, schema []model.TagDefinition) error {
	_, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"tag_schema": schema, "updated_at": time.Now()}})
	return err
}

// UpdateScrapeScript 设置/替换刮削脚本
func (s *CollectionStore) UpdateScrapeScript(ctx context.Context, id primitive.ObjectID, script *model.ScrapeScript) error {
	_, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"scrape_script": script, "updated_at": time.Now()}})
	return err
}

// AddMember 追加成员（原子防重：members.user_id 已存在时不生效）。
// 返回 mongo.ErrNoDocuments 表示集合不存在或用户已是成员。
func (s *CollectionStore) AddMember(ctx context.Context, id primitive.ObjectID, m model.Member) error {
	res, err := s.coll.UpdateOne(ctx,
		bson.M{"_id": id, "members.user_id": bson.M{"$ne": m.UserID}},
		bson.M{"$push": bson.M{"members": m}, "$set": bson.M{"updated_at": time.Now()}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// RemoveMember 移除成员
func (s *CollectionStore) RemoveMember(ctx context.Context, id, userID primitive.ObjectID) error {
	_, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$pull": bson.M{"members": bson.M{"user_id": userID}}, "$set": bson.M{"updated_at": time.Now()}})
	return err
}

// ReplaceAdmins 更换集合管理员：保留非管理员成员，管理员替换为 newAdminID
func (s *CollectionStore) ReplaceAdmins(ctx context.Context, id, newAdminID primitive.ObjectID) error {
	c, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return mongo.ErrNoDocuments
	}
	members := make([]model.Member, 0, len(c.Members)+1)
	for _, m := range c.Members {
		if m.Role == model.MemberRoleCollectionAdmin {
			continue
		}
		if m.UserID == newAdminID {
			continue // 原操作工升级为管理员，稍后以管理员身份加入
		}
		members = append(members, m)
	}
	members = append(members, model.Member{UserID: newAdminID, Role: model.MemberRoleCollectionAdmin})
	_, err = s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"members": members, "updated_at": time.Now()}})
	return err
}

// Delete 删除集合
func (s *CollectionStore) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := s.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateDeletePolicy 设置集合删除策略
func (s *CollectionStore) UpdateDeletePolicy(ctx context.Context, id primitive.ObjectID, policy *model.DeletePolicy) error {
	_, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"delete_policy": policy, "updated_at": time.Now()}})
	return err
}

// Count 统计集合数（支持任意过滤条件）
func (s *CollectionStore) Count(ctx context.Context, filter bson.M) (int64, error) {
	return s.coll.CountDocuments(ctx, filter)
}

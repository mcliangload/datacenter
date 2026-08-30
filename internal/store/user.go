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

// UserStore 用户数据访问
type UserStore struct {
	coll *mongo.Collection
}

// NewUserStore 构造用户存储
func NewUserStore(db *mongo.Database) *UserStore {
	return &UserStore{coll: db.Collection(model.CollectionUsers)}
}

// FindByUsername 按用户名查询，不存在时返回 (nil, nil)
func (s *UserStore) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := s.coll.FindOne(ctx, bson.M{"username": username}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByID 按 ID 查询，不存在时返回 (nil, nil)
func (s *UserStore) FindByID(ctx context.Context, id primitive.ObjectID) (*model.User, error) {
	var u model.User
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Create 创建用户；用户名重复时返回重复键错误
func (s *UserStore) Create(ctx context.Context, u *model.User) error {
	u.ID = primitive.NewObjectID()
	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	_, err := s.coll.InsertOne(ctx, u)
	return err
}

// List 分页查询用户，按创建时间倒序
func (s *UserStore) List(ctx context.Context, filter bson.M, page, pageSize int64) ([]*model.User, int64, error) {
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

	users := make([]*model.User, 0, pageSize)
	for cursor.Next(ctx) {
		var u model.User
		if err := cursor.Decode(&u); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}
	return users, total, cursor.Err()
}

// UpdateFields 按 ID 更新字段（自动刷新 updated_at）
func (s *UserStore) UpdateFields(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
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

// CountByRole 统计指定全局角色的用户数
func (s *UserStore) CountByRole(ctx context.Context, role string) (int64, error) {
	return s.coll.CountDocuments(ctx, bson.M{"role": role})
}

// Delete 删除用户
func (s *UserStore) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := s.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

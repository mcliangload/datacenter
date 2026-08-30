package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"datacenter/internal/logger"
	"datacenter/internal/model"
)

// AuditStore 审计日志存储
type AuditStore struct {
	coll *mongo.Collection
}

// NewAuditStore 构造审计日志存储
func NewAuditStore(db *mongo.Database) *AuditStore {
	return &AuditStore{coll: db.Collection(model.CollectionAuditLogs)}
}

// Log 记录审计日志（尽力而为，失败仅记日志，不影响主流程）
func (s *AuditStore) Log(ctx context.Context, actorID primitive.ObjectID, action, detail string, collectionID, itemID *primitive.ObjectID) {
	_, err := s.coll.InsertOne(ctx, &model.AuditLog{
		ActorID:      actorID,
		Action:       action,
		CollectionID: collectionID,
		ItemID:       itemID,
		Detail:       detail,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		logger.L().Warn("写入审计日志失败",
			zap.String("action", action),
			zap.Error(err))
	}
}

// List 分页查询审计日志（按时间倒序；系统优化 3.1：审计查询）
func (s *AuditStore) List(ctx context.Context, filter bson.M, page, pageSize int64) ([]*model.AuditLog, int64, error) {
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

	logs := make([]*model.AuditLog, 0, pageSize)
	for cursor.Next(ctx) {
		var l model.AuditLog
		if err := cursor.Decode(&l); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, total, cursor.Err()
}

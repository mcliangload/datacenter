package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"datacenter/internal/config"
	"datacenter/internal/logger"
	"datacenter/internal/model"
)

// DB 持有 MongoDB 客户端与数据库句柄
type DB struct {
	Client *mongo.Client
	DB     *mongo.Database
}

// auditRetentionDays 审计日志保留天数（系统优化 3.1，可经 DATACENTER_AUDIT_RETENTION_DAYS 覆盖）
var auditRetentionDays = func() int {
	if v := os.Getenv("DATACENTER_AUDIT_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 180
}()

// Connect 连接 MongoDB 并完成一次连通性探测
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("创建 MongoDB 客户端失败: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("MongoDB 连通性检测失败: %w", err)
	}
	logger.L().Info("MongoDB 连接成功",
		zap.String("uri", cfg.URI),
		zap.String("db", cfg.Name))
	return &DB{Client: client, DB: client.Database(cfg.Name)}, nil
}

// Close 关闭连接
func (d *DB) Close(ctx context.Context) error {
	return d.Client.Disconnect(ctx)
}

// EnsureIndexes 创建业务索引（幂等，可重复执行）
func EnsureIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	indexes := map[string][]mongo.IndexModel{
		model.CollectionUsers: {
			{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		model.CollectionBusiness: {
			// 集合名全局唯一（Q11）
			{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
			// 按成员列出集合
			{Keys: bson.D{{Key: "members.user_id", Value: 1}}},
		},
		model.CollectionDataItems: {
			{Keys: bson.D{{Key: "collection_id", Value: 1}, {Key: "created_at", Value: -1}}},
			// 同集合内路径唯一
			{Keys: bson.D{{Key: "collection_id", Value: 1}, {Key: "path", Value: 1}}, Options: options.Index().SetUnique(true)},
			// 物化路径：子树单查询（P3 优化）
			{Keys: bson.D{{Key: "ancestors", Value: 1}}},
			// 标签通配符索引（系统优化 2.1）：DQL 标签过滤（tags.xxx）不再集合内全扫描；
			// 存储开销随标签数量增长，大数据量下用 explain 基准评估（见 系统优化方案.md §2.1）
			{Keys: bson.D{{Key: "tags.$**", Value: 1}}},
		},
		model.CollectionScrapeTasks: {
			// Worker 领取任务与僵死回收
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}}},
			{Keys: bson.D{{Key: "item_id", Value: 1}, {Key: "created_at", Value: -1}}},
		},
		model.CollectionRelations: {
			// 出边分页查询
			{Keys: bson.D{{Key: "from_item_id", Value: 1}, {Key: "type", Value: 1}, {Key: "created_at", Value: -1}}},
			// 入边分页查询（变更影响分析）
			{Keys: bson.D{{Key: "to_item_id", Value: 1}, {Key: "type", Value: 1}, {Key: "created_at", Value: -1}}},
			// 幂等防重
			{Keys: bson.D{{Key: "from_item_id", Value: 1}, {Key: "to_item_id", Value: 1}, {Key: "type", Value: 1}}, Options: options.Index().SetUnique(true)},
			// 单父约束兜底（Q1）：同一子节点只允许一条父子入边
			{Keys: bson.D{{Key: "to_item_id", Value: 1}},
				Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"type": model.RelationParentChild})},
			// 集合级联删除边
			{Keys: bson.D{{Key: "collection_id", Value: 1}}},
		},
		model.CollectionAuditLogs: {
			{Keys: bson.D{{Key: "created_at", Value: -1}}},
			// 审计保留策略（系统优化 3.1）：超过保留天数自动过期删除（默认 180 天，可经 env 覆盖）
			{Keys: bson.D{{Key: "created_at", Value: 1}},
				Options: options.Index().SetExpireAfterSeconds(int32(auditRetentionDays * 24 * 3600))},
		},
	}

	for coll, models := range indexes {
		for _, im := range models {
			if _, err := db.Collection(coll).Indexes().CreateOne(ctx, im); err != nil {
				return fmt.Errorf("创建索引失败 (%s): %w", coll, err)
			}
		}
	}
	logger.L().Info("MongoDB 索引初始化完成")
	return nil
}

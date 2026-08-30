package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"datacenter/internal/config"
	"datacenter/internal/logger"
	"datacenter/internal/model"
)

// EnsureBootstrapAdmin 首次启动时创建默认管理员。
// 仅当系统中不存在任何 admin 角色用户时执行；未配置用户名/密码时跳过。
func EnsureBootstrapAdmin(db *mongo.Database, cfg config.BootstrapConfig) error {
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := db.Collection(model.CollectionUsers)
	n, err := coll.CountDocuments(ctx, bson.M{"role": model.RoleAdmin})
	if err != nil {
		return fmt.Errorf("查询 admin 用户失败: %w", err)
	}
	if n > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}

	now := time.Now()
	_, err = coll.InsertOne(ctx, &model.User{
		Username:     cfg.AdminUsername,
		PasswordHash: string(hash),
		Role:         model.RoleAdmin,
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		// 并发启动时可能同时创建，唯一索引兜底
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("创建默认管理员失败: %w", err)
	}
	logger.L().Info("已创建默认管理员", zap.String("username", cfg.AdminUsername))
	return nil
}

package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 全局角色
const (
	RoleAdmin = "admin" // 系统管理员：公共权限（用户管理、创建/删除集合等）
	RoleUser  = "user"  // 普通用户：无公共权限，等待被授权进入集合
)

// 用户状态
const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

// User 用户
type User struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username        string             `bson:"username" json:"username"`
	PasswordHash    string             `bson:"password_hash" json:"-"`
	PasswordVersion int                `bson:"password_version" json:"password_version"` // 安全增强 P1-7：改密时 +1 吊销旧 token
	Role            string             `bson:"role" json:"role"`
	Status          string             `bson:"status" json:"status"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

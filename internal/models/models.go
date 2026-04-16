package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BaseModel 基础模型结构
type BaseModel struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	CreatedBy string             `json:"created_by" bson:"created_by"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedBy string             `json:"updated_by" bson:"updated_by"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// FieldType 字段类型
type FieldType string

const (
	FieldTypeInt    FieldType = "int"
	FieldTypeFloat  FieldType = "float"
	FieldTypeString FieldType = "string"
	FieldTypeList   FieldType = "list"
)

// FieldDefinition 字段定义结构
type FieldDefinition struct {
	BaseModel
	Module      string     `json:"module" bson:"module"`
	FieldName   string     `json:"field_name" bson:"field_name"`
	FieldType   FieldType  `json:"field_type" bson:"field_type"`
	Description string     `json:"description" bson:"description"`
	Constraints Constraints `json:"constraints" bson:"constraints"`
}

// Constraints 字段约束
type Constraints struct {
	Min        *float64 `json:"min,omitempty" bson:"min,omitempty"`
	Max        *float64 `json:"max,omitempty" bson:"max,omitempty"`
	MinLength  *int     `json:"min_length,omitempty" bson:"min_length,omitempty"`
	MaxLength  *int     `json:"max_length,omitempty" bson:"max_length,omitempty"`
	ListLength *int     `json:"list_length,omitempty" bson:"list_length,omitempty"`
}

// BusinessData 业务数据结构
type BusinessData struct {
	BaseModel
	Module       string                 `json:"module" bson:"module"`
	Description  string                 `json:"description" bson:"description"`
	CustomFields map[string]interface{} `json:"custom_fields" bson:"custom_fields"`
	FilePath     string                 `json:"file_path" bson:"file_path"`
}

// DeletedData 已删除数据结构
type DeletedData struct {
	BaseModel
	Module       string                 `json:"module" bson:"module"`
	OriginalID   primitive.ObjectID     `json:"original_id" bson:"original_id"`
	Description  string                 `json:"description" bson:"description"`
	CustomFields map[string]interface{} `json:"custom_fields" bson:"custom_fields"`
	FilePath     string                 `json:"file_path" bson:"file_path"`
	DeletedAt    time.Time              `json:"deleted_at" bson:"deleted_at"`
}

// User 用户模型
type User struct {
	BaseModel
	Username    string   `json:"username" bson:"username"`
	Password    string   `json:"password,omitempty" bson:"password"`
	Email       string   `json:"email" bson:"email"`
	Roles       []string `json:"roles" bson:"roles"`
	Permissions []string `json:"permissions" bson:"permissions"`
}

// Permission 权限模型
type Permission struct {
	BaseModel
	Name        string `json:"name" bson:"name"`
	Code        string `json:"code" bson:"code"`
	Description string `json:"description" bson:"description"`
}

// Role 角色模型
type Role struct {
	BaseModel
	Name        string   `json:"name" bson:"name"`
	Code        string   `json:"code" bson:"code"`
	Description string   `json:"description" bson:"description"`
	Permissions []string `json:"permissions" bson:"permissions"`
}

// RolePermission 角色权限关联模型
type RolePermission struct {
	BaseModel
	RoleID       string `json:"role_id" bson:"role_id"`
	PermissionID string `json:"permission_id" bson:"permission_id"`
}

// UserRole 用户角色关联模型
type UserRole struct {
	BaseModel
	UserID string `json:"user_id" bson:"user_id"`
	RoleID string `json:"role_id" bson:"role_id"`
}

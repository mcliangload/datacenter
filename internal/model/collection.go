package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 标签值类型
type TagType string

const (
	TagTypeString TagType = "string"
	TagTypeInt    TagType = "int"
	TagTypeFloat  TagType = "float"
	TagTypeBool   TagType = "bool"
	TagTypeDate   TagType = "date"
	TagTypeEnum   TagType = "enum"
	TagTypeArray  TagType = "array"
	TagTypeObject TagType = "object"
)

// 集合级成员角色
type MemberRole string

const (
	MemberRoleCollectionAdmin MemberRole = "collection_admin"
	MemberRoleOperator        MemberRole = "operator"
)

// TagDefinition 标签定义
type TagDefinition struct {
	Name        string          `bson:"name" json:"name"`
	Type        TagType         `bson:"type" json:"type"`
	Required    bool            `bson:"required" json:"required"`
	EnumValues  []string        `bson:"enum_values,omitempty" json:"enum_values,omitempty"`
	ElementType TagType         `bson:"element_type,omitempty" json:"element_type,omitempty"`
	Fields      []TagDefinition `bson:"fields,omitempty" json:"fields,omitempty"`
}

// Member 集合成员（集合级授权，逐集合独立定义）
type Member struct {
	UserID primitive.ObjectID `bson:"user_id" json:"user_id"`
	Role   MemberRole         `bson:"role" json:"role"`
}

// ScrapeScript 集合的刮削脚本（每个集合有且仅有一个，以 NFS 路径注册）
type ScrapeScript struct {
	Path      string             `bson:"path" json:"path"`
	UpdatedBy primitive.ObjectID `bson:"updated_by" json:"updated_by"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// BusinessCollection 业务集合
type BusinessCollection struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Description  string             `bson:"description" json:"description"`
	CreatedBy    primitive.ObjectID `bson:"created_by" json:"created_by"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	TagSchema    []TagDefinition    `bson:"tag_schema" json:"tag_schema"`
	ScrapeScript *ScrapeScript      `bson:"scrape_script,omitempty" json:"scrape_script,omitempty"`
	Members      []Member           `bson:"members" json:"members"`
	DeletePolicy *DeletePolicy      `bson:"delete_policy,omitempty" json:"delete_policy,omitempty"`
}

// EffectiveDeletePolicy 返回生效的删除策略（未配置时用默认：均拒绝）
func (c *BusinessCollection) EffectiveDeletePolicy() DeletePolicy {
	if c.DeletePolicy == nil {
		return DefaultDeletePolicy()
	}
	return *c.DeletePolicy
}

// RoleOf 返回用户在该集合中的角色；非成员返回 ok=false
func (c *BusinessCollection) RoleOf(userID primitive.ObjectID) (MemberRole, bool) {
	for _, m := range c.Members {
		if m.UserID == userID {
			return m.Role, true
		}
	}
	return "", false
}

// IsMember 是否为集合成员
func (c *BusinessCollection) IsMember(userID primitive.ObjectID) bool {
	_, ok := c.RoleOf(userID)
	return ok
}

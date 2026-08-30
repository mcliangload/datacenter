package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 关联关系类型
const (
	RelationParentChild = "parent_child" // 父子/包含：父拥有子（单父树，Q1）
	RelationReference   = "reference"    // 引用：弱引用，不拥有
	RelationCall        = "call"         // 调用：运行时依赖
)

// 集合删除策略取值（§7 方案）
const (
	PolicyDeny    = "deny"    // 拒绝（可逃生参数覆盖）
	PolicyCascade = "cascade" // 级联删除
	PolicyDetach  = "detach"  // 自动解除关系
)

// Relation 数据项关联边：from_item_id → to_item_id（有向）
type Relation struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	CollectionID primitive.ObjectID     `bson:"collection_id" json:"collection_id"` // 起点所属集合（冗余，集合级联删除用）
	FromItemID   primitive.ObjectID     `bson:"from_item_id" json:"from_item_id"`
	ToItemID     primitive.ObjectID     `bson:"to_item_id" json:"to_item_id"`
	Type         string                 `bson:"type" json:"type"`
	Meta         map[string]interface{} `bson:"meta,omitempty" json:"meta,omitempty"`
	CreatedBy    primitive.ObjectID     `bson:"created_by" json:"created_by"`
	CreatedAt    time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time              `bson:"updated_at" json:"updated_at"`
}

// DeletePolicy 集合删除策略（nil 表示使用默认值）
type DeletePolicy struct {
	Children string `bson:"children" json:"children"` // deny | cascade | detach
	Incoming string `bson:"incoming" json:"incoming"` // deny | detach
}

// DefaultDeletePolicy 默认策略：有子节点拒绝、被引用拒绝（最安全）
func DefaultDeletePolicy() DeletePolicy {
	return DeletePolicy{Children: PolicyDeny, Incoming: PolicyDeny}
}

// ValidDeletePolicy 校验策略取值
func ValidDeletePolicy(p DeletePolicy) bool {
	switch p.Children {
	case PolicyDeny, PolicyCascade, PolicyDetach:
	default:
		return false
	}
	switch p.Incoming {
	case PolicyDeny, PolicyDetach:
	default:
		return false
	}
	return true
}

package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuditLog 审计日志：敏感操作记录
type AuditLog struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ActorID      primitive.ObjectID  `bson:"actor_id" json:"actor_id"`
	Action       string              `bson:"action" json:"action"`
	CollectionID *primitive.ObjectID `bson:"collection_id,omitempty" json:"collection_id,omitempty"`
	ItemID       *primitive.ObjectID `bson:"item_id,omitempty" json:"item_id,omitempty"`
	Detail       string              `bson:"detail" json:"detail"`
	CreatedAt    time.Time           `bson:"created_at" json:"created_at"`
}

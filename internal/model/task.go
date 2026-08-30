package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 刮削任务状态
const (
	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
)

// ScrapeTask 刮削任务：对数据项执行集合的刮削脚本并产出标签
type ScrapeTask struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	CollectionID primitive.ObjectID     `bson:"collection_id" json:"collection_id"`
	ItemID       primitive.ObjectID     `bson:"item_id" json:"item_id"`
	ScriptPath   string                 `bson:"script_path" json:"script_path"` // 执行时的脚本路径快照
	DataPath     string                 `bson:"data_path" json:"data_path"`     // 执行时的数据路径快照
	Status       string                 `bson:"status" json:"status"`
	TriggerBy    string                 `bson:"trigger_by" json:"trigger_by"` // auto 或 user_id hex
	CreatedAt    time.Time              `bson:"created_at" json:"created_at"`
	StartedAt    *time.Time             `bson:"started_at,omitempty" json:"started_at,omitempty"`
	FinishedAt   *time.Time             `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
	ExitCode     *int                   `bson:"exit_code,omitempty" json:"exit_code,omitempty"`
	Error        string                 `bson:"error,omitempty" json:"error,omitempty"`
	ResultTags   map[string]interface{} `bson:"result_tags,omitempty" json:"result_tags,omitempty"`
	RetryOf      *primitive.ObjectID    `bson:"retry_of,omitempty" json:"retry_of,omitempty"`
}

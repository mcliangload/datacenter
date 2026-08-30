package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 数据项标签来源
const (
	TagSourceManual = "manual"
	TagSourceScrape = "scrape"
	TagSourceMixed  = "mixed"
)

// 数据项刮削状态
const (
	ItemScrapeNone    = "none"    // 未刮削
	ItemScrapePending = "pending" // 刮削中/等待刮削
	ItemScrapeSuccess = "success" // 刮削成功
	ItemScrapeFailed  = "failed"  // 刮削失败
)

// DataItem 数据项：NFS 上的文件/文件夹路径 + 标签值
// 标签优先级：手动标签始终优先（manual_tags 持久化操作工录入的标签），
// 刮削结果仅补充手动未产出的标签；tags 为合并后的有效视图。
type DataItem struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	CollectionID  primitive.ObjectID     `bson:"collection_id" json:"collection_id"`
	Path          string                 `bson:"path" json:"path"`
	Tags          map[string]interface{} `bson:"tags,omitempty" json:"tags,omitempty"`               // 有效标签（合并视图）
	ManualTags    map[string]interface{} `bson:"manual_tags,omitempty" json:"manual_tags,omitempty"` // 手动标签（始终优先）
	Ancestors     []primitive.ObjectID   `bson:"ancestors,omitempty" json:"ancestors,omitempty"`     // 物化路径：父链（P3 优化，子树单查询）
	TagSource     string                 `bson:"tag_source" json:"tag_source"`
	ScrapeStatus  string                 `bson:"scrape_status" json:"scrape_status"`
	LastScrapedAt *time.Time             `bson:"last_scraped_at,omitempty" json:"last_scraped_at,omitempty"`
	CreatedBy     primitive.ObjectID     `bson:"created_by" json:"created_by"`
	CreatedAt     time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time              `bson:"updated_at" json:"updated_at"`
}

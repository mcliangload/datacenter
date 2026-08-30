package model

// MongoDB 集合名（存储层常量，注意与「业务集合」概念区分）
const (
	CollectionUsers       = "users"                // 用户
	CollectionBusiness    = "business_collections" // 业务集合
	CollectionDataItems   = "data_items"           // 数据项
	CollectionScrapeTasks = "scrape_tasks"         // 刮削任务
	CollectionAuditLogs   = "audit_logs"           // 审计日志
	CollectionRelations   = "data_relations"       // 数据项关联边
)

# 刮削系统模块 - 技术文档

## 1. 模块概述

刮削（Scrape）系统是异步任务处理引擎，基于 Go 协程池 + Channel 队列架构，用于执行外部数据刮削器脚本，将刮削结果自动存入业务数据库。

### 模块位置

```
internal/scraper/
└── scraper.go           # 刮削系统核心实现

internal/models/models.go  # ScrapeTask, DeletedScrapeTask 模型
internal/storage/mongodb_storage.go  # 刮削任务存储层
internal/api/handlers.go  # 刮削任务 API handler
```

---

## 2. 接口设计

### 2.1 Scraper 接口

```go
type Scraper interface {
    SubmitTask(task *models.ScrapeTask) error
    Start() error
    Stop() error
}
```

### 2.2 实现结构体

```go
type scraper struct {
    storage    storage.Storage
    taskQueue  chan *models.ScrapeTask  // 缓冲区大小 1000
    workerPool int                      // 工作协程数，默认 4
    stopChan   chan struct{}            // 停止信号
}
```

### 2.3 构造函数

```go
func NewScraper(storage storage.Storage, workerPool int) Scraper
```

| 参数 | 说明 |
|------|------|
| `storage` | 业务数据存储接口 |
| `workerPool` | 工作协程数，≤0 则默认 4 |

---

## 3. 数据模型

### 3.1 ScrapeTask

```go
type ScrapeTask struct {
    ID             primitive.ObjectID `json:"_id" bson:"_id"`
    Module         string             `json:"module" bson:"module"`
    DataPath       string             `json:"data_path" bson:"data_path"`
    ScraperPath    string             `json:"scraper_path" bson:"scraper_path"`
    Status         ScrapeTaskStatus   `json:"status" bson:"status"`
    Result         interface{}        `json:"result" bson:"result"`
    ErrorMessage   string             `json:"error_message" bson:"error_message"`
    StartedAt      *time.Time         `json:"started_at" bson:"started_at"`
    CompletedAt    *time.Time         `json:"completed_at" bson:"completed_at"`
    BusinessDataID primitive.ObjectID `json:"business_data_id,omitempty" bson:"business_data_id,omitempty"`
    Description    string             `json:"description" bson:"description"`
    BaseModel
}
```

### 3.2 任务状态

```go
type ScrapeTaskStatus string

const (
    ScrapeTaskStatusPending  ScrapeTaskStatus = "pending"
    ScrapeTaskStatusScraping ScrapeTaskStatus = "scraping"
    ScrapeTaskStatusSuccess  ScrapeTaskStatus = "success"
    ScrapeTaskStatusFailed   ScrapeTaskStatus = "failed"
)
```

---

## 4. 核心流程

### 4.1 启动

```go
func (s *scraper) Start() error
```

1. 启动 `workerPool` 个 goroutine，每个执行 `worker(id)` 循环
2. 每个 worker 在 `for-select` 中监听 `taskQueue` 和 `stopChan`

### 4.2 提交任务

```go
func (s *scraper) SubmitTask(task *models.ScrapeTask) error
```

1. **参数验证**: 检查 module/dataPath/scraperPath 非空
2. **模块验证**: 调用 `storage.GetCollectionByModule(task.Module)` 确认模块存在
3. **持久化**: `storage.CreateScrapeTask(task)` 写入 MongoDB，状态为 `pending`
4. **入队**: `s.taskQueue <- task`

### 4.3 Worker 处理

```go
func (s *scraper) processTask(task *models.ScrapeTask, workerID int)
```

```
1. 更新状态 → scraping，记录 started_at
      │
      ▼
2. executeScraper(scraperPath, dataPath)
   ├── os.Stat 检查文件存在
   ├── exec.Command("python", scraperPath, dataPath)
   └── JSON 解析输出: { success, data, error }
      │
      ├── 成功 ──▶ 3. saveScrapedData()
      │             ├── 合并刮削属性
      │             ├── 字段验证（非阻塞）
      │             └── 存入 {module}_data
      │
      └── 失败 ──▶ 记录 error_message
      │
      ▼
4. 更新最终状态 → success / failed + completed_at + business_data_id
```

### 4.4 刮削器执行

```go
func (s *scraper) executeScraper(scraperPath, dataPath string) (map[string]interface{}, error)
```

**刮削器脚本规范**:
- 脚本接收两个参数: `scraperPath` 和 `dataPath`
- 输出 JSON 到 stdout: `{ "success": true/false, "data": {...}, "error": "..." }`
- Python 示例: `python movie_scraper.py /data/movies/hp.json`

### 4.5 结果存储

```go
func (s *scraper) saveScrapedData(task *models.ScrapeTask, data map[string]interface{}) (primitive.ObjectID, error)
```

存储时自动添加以下元数据字段：
- `scrape_path`: 刮削器路径
- `data_path`: 源数据路径
- `module`: 所属模块
- `task_id`: 刮削任务 ID
- `scraped_at`: 刮削完成时间

---

## 5. API 接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | /api/scraper/upload | scrape:write | 提交刮削任务 |
| GET | /api/scraper/tasks | scrape:read | 任务列表（分页，支持 module/status 过滤） |
| GET | /api/scraper/tasks/:id | scrape:read | 任务详情 |
| POST | /api/scraper/tasks/:id/retry | scrape:write | 重试任务 |
| DELETE | /api/scraper/tasks/:id | scrape:write | 软删除单个任务 |
| POST | /api/scraper/tasks/batch-delete | scrape:write | 批量软删除 |
| GET | /api/deleted-scraper/module/:module | scrape:read | 已删除任务列表 |
| GET | /api/deleted-scraper/:id | scrape:read | 已删除任务详情 |
| POST | /api/deleted-scraper/:id/recover | scrape:write | 恢复删除的任务 |

---

## 6. 任务重试

```go
func (h *Handler) RetryScrapeTask(c *gin.Context)
```

1. 获取任务记录
2. 可选更新 `scraperPath`
3. 重置状态为 `pending`，清除 `error_message`、`started_at`、`completed_at`
4. 重新 `SubmitTask(task)`

---

## 7. 停止流程

```go
func (s *scraper) Stop() error
```

1. `close(s.stopChan)` → 所有 worker 的 `select` 收到信号，退出循环
2. `close(s.taskQueue)` → 队列关闭，worker 处理完当前任务后退出

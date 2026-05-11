# 刮削系统模块 - 需求实现文档

## 1. 实现概述

刮削系统采用经典的 **Worker Pool + Channel 队列** 架构，核心实现在 `internal/scraper/scraper.go`，约 290 行代码。

---

## 2. 文件清单

| 文件 | 说明 |
|------|------|
| `internal/scraper/scraper.go` | 刮削系统核心：Scraper 接口、worker pool、任务处理 |
| `internal/models/models.go` | ScrapeTask、DeletedScrapeTask 模型 |
| `internal/storage/mongodb_storage.go` | 刮削任务存储层 |
| `internal/api/handlers.go` | 刮削 API handler |
| `cmd/server/main.go` | 刮削系统初始化与生命周期管理 |

---

## 3. 系统初始化与生命周期

### 3.1 启动

```go
// cmd/server/main.go:65-70
scraperSystem := scraper.NewScraper(
    businessStorage,
    getEnvAsInt("SCRAPER_WORKERS", 4),
)
if err := scraperSystem.Start(); err != nil {
    panic("Failed to start scraper system: " + err.Error())
}
defer scraperSystem.Stop()
```

### 3.2 优雅关闭

```go
// 在信号处理中
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
scraperSystem.Stop()  // close(stopChan) + close(taskQueue)
```

---

## 4. Worker Pool 实现

### 4.1 启动 Workers

```go
// internal/scraper/scraper.go:48-57
func (s *scraper) Start() error {
    log.Info().Msg("启动刮削系统")
    for i := 0; i < s.workerPool; i++ {
        go s.worker(i)
    }
    log.Info().Int("workers", s.workerPool).Msg("刮削系统启动完成")
    return nil
}
```

### 4.2 Worker 循环

```go
// internal/scraper/scraper.go:102-120
func (s *scraper) worker(id int) {
    log.Info().Int("worker_id", id).Msg("工作协程启动")

    for {
        select {
        case task, ok := <-s.taskQueue:
            if !ok {
                // taskQueue 已关闭，worker 退出
                log.Info().Int("worker_id", id).Msg("任务队列已关闭，工作协程退出")
                return
            }
            s.processTask(task, id)

        case <-s.stopChan:
            // 收到停止信号
            log.Info().Int("worker_id", id).Msg("收到停止信号，工作协程退出")
            return
        }
    }
}
```

**设计要点**:
- 使用 `for-select` 模式同时监听任务队列和停止信号
- `taskQueue` 关闭时 `ok == false`，worker 安全退出
- `stopChan` 关闭时所有 worker 同时收到信号

---

## 5. 任务提交流程

```go
// internal/scraper/scraper.go:75-99
func (s *scraper) SubmitTask(task *models.ScrapeTask) error {
    // 1. 参数验证
    if task.Module == "" || task.DataPath == "" || task.ScraperPath == "" {
        return fmt.Errorf("无效的刮削任务参数")
    }

    // 2. 模块存在性检查
    _, err := s.storage.GetCollectionByModule(task.Module)
    if err != nil {
        return fmt.Errorf("模块不存在: %s，请先创建模块集合", task.Module)
    }

    // 3. 持久化到 MongoDB
    err = s.storage.CreateScrapeTask(task)
    if err != nil {
        return fmt.Errorf("保存刮削任务失败: %v", err)
    }

    // 4. 入队
    s.taskQueue <- task
    log.Info().Str("task_id", task.ID.Hex()).Str("module", task.Module).Msg("刮削任务已提交")

    return nil
}
```

---

## 6. 任务处理核心

### 6.1 processTask

```go
// internal/scraper/scraper.go:123-193
func (s *scraper) processTask(task *models.ScrapeTask, workerID int) {
    // 1. 更新状态 → scraping
    task.Status = models.ScrapeTaskStatusScraping
    now := time.Now()
    task.StartedAt = &now
    s.storage.UpdateScrapeTask(task)

    // 2. 执行刮削器
    result, err := s.executeScraper(task.ScraperPath, task.DataPath)

    if err != nil {
        // 3a. 失败处理
        task.Status = models.ScrapeTaskStatusFailed
        task.ErrorMessage = err.Error()
        completedNow := time.Now()
        task.CompletedAt = &completedNow
        s.storage.UpdateScrapeTask(task)
        return
    }

    // 3b. 成功处理
    task.Status = models.ScrapeTaskStatusSuccess
    task.Result = result
    completedNow := time.Now()
    task.CompletedAt = &completedNow

    // 4. 存储结果到业务数据
    dataID, err := s.saveScrapedData(task, result)
    if err != nil {
        // 存储失败也更新任务状态
        log.Error().Msg("存储刮削结果失败")
    } else {
        task.BusinessDataID = dataID
    }

    s.storage.UpdateScrapeTask(task)
}
```

### 6.2 executeScraper

```go
// internal/scraper/scraper.go:196-231
func (s *scraper) executeScraper(scraperPath, dataPath string) (map[string]interface{}, error) {
    // 1. 文件存在性检查
    if _, err := os.Stat(scraperPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("刮削器文件不存在: %s", scraperPath)
    }
    if _, err := os.Stat(dataPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("数据文件不存在: %s", dataPath)
    }

    // 2. 执行外部脚本
    cmd := exec.Command("python", scraperPath, dataPath)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("执行刮削器失败: %v, 输出: %s", err, output)
    }

    // 3. 解析 JSON 输出
    var result struct {
        Success bool                   `json:"success"`
        Data    map[string]interface{} `json:"data"`
        Error   string                 `json:"error"`
    }
    err = json.Unmarshal(output, &result)
    if err != nil {
        return nil, fmt.Errorf("解析刮削器输出失败: %v, 输出: %s", err, output)
    }

    if !result.Success {
        return nil, fmt.Errorf("刮削器执行失败: %s", result.Error)
    }

    return result.Data, nil
}
```

**刮削器输出契约**:
```json
{
  "success": true,
  "data": {
    "title": "Harry Potter",
    "year": 2001,
    "rating": 7.6
  },
  "error": ""
}
```

### 6.3 saveScrapedData

```go
// internal/scraper/scraper.go:234-288
func (s *scraper) saveScrapedData(task *models.ScrapeTask, data map[string]interface{}) (primitive.ObjectID, error) {
    // 1. 合并刮削元数据
    enhancedData := make(map[string]interface{})
    for k, v := range data {
        enhancedData[k] = v
    }
    enhancedData["scrape_path"] = task.ScraperPath
    enhancedData["data_path"] = task.DataPath
    enhancedData["module"] = task.Module
    enhancedData["task_id"] = task.ID.Hex()
    enhancedData["scraped_at"] = time.Now()

    // 2. 字段验证（非阻塞，告警但继续保存）
    fieldDefs, _ := s.storage.GetFieldDefinitionsByModule(task.Module)
    for _, fieldDef := range fieldDefs {
        value := enhancedData[fieldDef.FieldName]
        result := fieldDef.Validate(value)
        if !result.Valid {
            log.Warn().Str("field", fieldDef.FieldName).
                Interface("errors", result.Errors).
                Msg("刮削结果字段验证失败，将使用原始值保存")
        }
    }

    // 3. 构造 BusinessData
    businessData := &models.BusinessData{
        Module:       task.Module,
        Description:  fmt.Sprintf("刮削数据 - %s", task.DataPath),
        CustomFields: enhancedData,
        FilePath:     task.DataPath,
        BaseModel: models.BaseModel{
            CreatedBy: task.CreatedBy,
            CreatedAt: time.Now(),
        },
    }

    // 4. 存入动态集合
    collectionName := task.Module + "_data"
    err := s.storage.CreateBusinessData(context.Background(), collectionName, businessData)
    if err != nil {
        return primitive.NilObjectID, fmt.Errorf("存储业务数据失败: %v", err)
    }

    return businessData.ID, nil
}
```

**关键设计**: 字段验证失败只是告警（`log.Warn`），不阻止数据存储。这确保了即使字段定义变更，已刮削的数据仍可成功保存。

---

## 7. API Handler 实现

### 7.1 提交任务

```go
// internal/api/handlers.go:1189-1226
func (h *Handler) SubmitScrapeTask(c *gin.Context) {
    var req struct {
        Module      string `json:"module" binding:"required"`
        DataPath    string `json:"data_path" binding:"required"`
        ScraperPath string `json:"scraper_path" binding:"required"`
        Description string `json:"description"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    task := &models.ScrapeTask{
        Module:      req.Module,
        DataPath:    req.DataPath,
        ScraperPath: req.ScraperPath,
        Status:      models.ScrapeTaskStatusPending,
        Description: req.Description,
    }
    // 记录操作人
    if userID, exists := c.Get("user_id"); exists {
        task.CreatedBy = userID.(string)
    }

    if err := h.scraper.SubmitTask(task); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Scrape task submitted successfully",
        "task_id": task.ID,
    })
}
```

### 7.2 重试任务

```go
// internal/api/handlers.go:1266-1301
func (h *Handler) RetryScrapeTask(c *gin.Context) {
    id := c.Param("id")
    task, _ := h.storage.GetScrapeTaskByID(id)

    // 可选更新 scraper_path
    var req struct{ ScraperPath string `json:"scraper_path"` }
    if c.ShouldBindJSON(&req) == nil && req.ScraperPath != "" {
        task.ScraperPath = req.ScraperPath
    }

    // 重置状态
    task.Status = models.ScrapeTaskStatusPending
    task.ErrorMessage = ""
    task.StartedAt = nil
    task.CompletedAt = nil

    h.scraper.SubmitTask(task)
    c.JSON(200, gin.H{"message": "Scrape task retry submitted", "task_id": task.ID})
}
```

### 7.3 批量删除

```go
// internal/api/handlers.go:1314-1332
func (h *Handler) BatchDeleteScrapeTasks(c *gin.Context) {
    var req struct{ IDs []string `json:"ids" binding:"required"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    deletedCount := 0
    for _, id := range req.IDs {
        if err := h.storage.DeleteScrapeTask(id); err != nil {
            continue  // 跳过失败的，不中断批量操作
        }
        deletedCount++
    }
    c.JSON(200, gin.H{"message": "...", "deleted_count": deletedCount})
}
```

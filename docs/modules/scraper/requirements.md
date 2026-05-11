# 刮削系统模块 - 需求设计文档

## 1. 需求背景

数据中心需要从外部数据源（文件、API 等）通过刮削器脚本提取结构化数据并存入系统。刮削过程是异步、可能耗时的操作，需要一个可靠的任务调度系统来管理这些刮削任务。

## 2. 功能需求

### FR-SCR-01: 提交刮削任务

用户提交一个刮削任务，指定数据源和刮削器。

**输入**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 目标模块（数据存入哪个集合） |
| data_path | string | 是 | 源数据文件路径 |
| scraper_path | string | 是 | 刮削器脚本路径 |
| description | string | 否 | 任务描述 |

**规则**:
- 模块集合必须存在
- 任务创建后立即入队，异步执行

### FR-SCR-02: 任务状态跟踪

**4 种状态**:
| 状态 | 说明 |
|------|------|
| pending | 已提交，等待处理 |
| scraping | 正在执行刮削器 |
| success | 刮削成功，结果已存储 |
| failed | 刮削失败，记录错误信息 |

**状态变更**:
- 提交 → pending
- Worker 取出 → scraping（记录 started_at）
- 执行完成 → success（记录 completed_at + result + business_data_id）
- 执行失败 → failed（记录 completed_at + error_message）

### FR-SCR-03: 查看任务

| 操作 | 说明 |
|------|------|
| 任务列表 | 分页查询，支持按 module 和 status 过滤 |
| 任务详情 | 按 ID 查看完整信息 |

### FR-SCR-04: 任务重试

失败的任务可以重新提交执行。

**规则**:
- 可修改 scraper_path
- 重置状态为 pending
- 清除之前的 error_message、started_at、completed_at

### FR-SCR-05: 任务删除与恢复

- 删除操作：软删除（移到 deleted_scrape_tasks）
- 恢复操作：从已删除列表恢复到 scrape_tasks

### FR-SCR-06: 批量删除

支持一次删除多个任务（传入 ID 数组）。

### FR-SCR-07: 刮削结果自动存储

刮削成功后，结果数据自动存入对应模块的业务数据集合（`{module}_data`），并关联任务 ID。

---

## 3. 架构设计

### 3.1 Worker Pool 模式

```
                        ┌─────────────────┐
   API Handler ────────▶│   taskQueue     │
   SubmitTask()         │ chan *ScrapeTask│
                        │ (buffer = 1000) │
                        └────────┬────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
        ┌──────────┐      ┌──────────┐      ┌──────────┐
        │ Worker 0 │      │ Worker 1 │ ...  │ Worker N │
        │ goroutine│      │ goroutine│      │ goroutine│
        └──────────┘      └──────────┘      └──────────┘
```

### 3.2 并发参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| SCRAPER_WORKERS | 4 | 工作协程数 |
| 队列缓冲区 | 1000 | Channel 缓冲区大小 |

---

## 4. 非功能需求

| 编号 | 需求 | 说明 |
|------|------|------|
| NFR-SCR-01 | 异步执行 | 任务提交后立即返回，不阻塞 HTTP 请求 |
| NFR-SCR-02 | 并发处理 | 支持多个任务同时执行 |
| NFR-SCR-03 | 优雅关闭 | 服务关闭时等待当前任务完成 |
| NFR-SCR-04 | 错误隔离 | 单个任务失败不影响其他任务 |

---

## 5. 业务流程

### 5.1 提交并执行刮削

```
POST /api/scraper/upload { module, data_path, scraper_path }
      │
      ▼
JWT 认证 + scrape:write 权限检查
      │
      ▼
验证 module 集合存在
      │
      ▼
创建 ScrapeTask (status=pending) → 存入 scrape_tasks
      │
      ▼
taskQueue <- task
      │
      ▼
立即返回 { message, task_id }
      │
      │ (异步)
      ▼
Worker 从 taskQueue 取出任务
      │
      ▼
更新状态 → scraping, 记录 started_at
      │
      ▼
executeScraper:
  ├── os.Stat(scraperPath) → 文件存在?
  ├── os.Stat(dataPath) → 文件存在?
  ├── exec.Command("python", scraperPath, dataPath)
  └── JSON 解析 stdout
      │
      ├── 成功 ──▶ saveScrapedData → 存入 movie_data
      │              └── 更新 status=success, business_data_id
      │
      └── 失败 ──▶ 更新 status=failed, error_message
```

### 5.2 重试任务

```
POST /api/scraper/tasks/:id/retry { scraper_path? }
      │
      ▼
获取任务记录
      │
      ▼
可选的 scraper_path 更新
      │
      ▼
重置: status=pending, error_message="", started_at=nil, completed_at=nil
      │
      ▼
SubmitTask(task) → 重新入队
```

## 6. 接口定义

### 提交任务

```json
POST /api/scraper/upload
Request:
{
  "module": "movie",
  "data_path": "/data/movies/harry_potter.json",
  "scraper_path": "/scrapers/movie_scraper.py",
  "description": "刮削哈利波特电影数据"
}
Response 200:
{
  "message": "Scrape task submitted successfully",
  "task_id": "507f1f77bcf86cd799439011"
}
```

### 任务列表

```json
GET /api/scraper/tasks?module=movie&status=success&page=1&pageSize=10
Response 200:
{
  "data": [
    {
      "_id": "...",
      "module": "movie",
      "data_path": "/data/movies/hp.json",
      "status": "success",
      "result": { "title": "Harry Potter", ... },
      "started_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:30:02Z",
      "business_data_id": "..."
    }
  ],
  "total": 50,
  "page": 1,
  "pageSize": 10
}
```

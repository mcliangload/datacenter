# 日志系统

## 1. 日志等级

系统使用 zerolog 实现结构化日志，支持以下日志等级：

| 等级 | 描述 | 用途 |
|------|------|------|
| debug | 调试级别 | 开发调试信息 |
| info | 信息级别 | 一般运行信息 |
| warn | 警告级别 | 警告信息 |
| error | 错误级别 | 错误信息 |

日志等级可通过环境变量 `LOG_LEVEL` 配置，默认为 `info`。

## 2. 日志类型

### 2.1 应用日志 (Logger)

- 同时输出到标准输出和控制台文件
- 文件路径：`logs/app.log`
- 包含调用者信息（caller）

### 2.2 HTTP日志 (HTTPLogger)

- 仅输出到HTTP日志文件
- 文件路径：`logs/http.log`（可通过 `LOG_HTTP_FILE` 配置）
- 记录所有HTTP请求和响应信息

## 3. 日志中间件

### 3.1 Gin日志中间件

HTTP日志中间件记录以下请求信息：
- 请求方法（method）
- 请求路径（path）
- 响应状态码（status）
- 请求处理时间（latency）
- 客户端IP（client_ip）
- 用户代理（user_agent）

## 4. 日志文件管理

### 4.1 文件配置

使用 lumberjack 进行日志轮转：

| 参数 | 默认值 | 环境变量 | 说明 |
|------|--------|----------|------|
| MaxSize | 100MB | LOG_MAX_SIZE | 单个日志文件最大大小 |
| MaxBackups | 5 | LOG_MAX_BACKUPS | 保留日志文件数 |
| MaxAge | 30 | LOG_MAX_AGE | 日志保留天数 |
| Compress | true | - | 是否压缩旧日志 |

### 4.2 日志目录结构

```
logs/
├── http.log      # HTTP请求日志
└── app.log       # 应用日志
```

## 5. 日志格式

### 5.1 应用日志格式

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "caller": "scraper.go:123",
  "message": "刮削任务处理完成"
}
```

### 5.2 HTTP日志格式

```json
{
  "level": "info",
  "time": "2024-01-15T10:00:00Z",
  "method": "POST",
  "path": "/api/scraper/upload",
  "status": 200,
  "latency": "45ms",
  "client_ip": "192.168.1.1"
}
```

## 6. 日志配置

### 6.1 环境变量配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| LOG_LEVEL | info | 日志级别 |
| LOG_HTTP_FILE | logs/http.log | HTTP日志文件路径 |
| LOG_MAX_SIZE | 100 | 单个日志文件最大MB |
| LOG_MAX_BACKUPS | 5 | 保留日志文件数 |
| LOG_MAX_AGE | 30 | 日志保留天数 |

### 6.2 初始化参数

```go
logger.Init(
    getEnv("LOG_LEVEL", "info"),           // 日志级别
    getEnv("LOG_HTTP_FILE", "logs/http.log"), // HTTP日志文件
    getEnvAsInt("LOG_MAX_SIZE", 100),      // 最大文件大小(MB)
    getEnvAsInt("LOG_MAX_BACKUPS", 5),     // 保留备份数
    getEnvAsInt("LOG_MAX_AGE", 30),        // 保留天数
)
```

## 7. 日志记录点

系统在以下关键节点记录日志：

### 7.1 系统启动

- 环境变量加载完成
- 日志初始化完成
- 存储初始化完成
- RBAC存储初始化完成
- 默认数据初始化完成
- 刮削系统启动完成
- JWT服务初始化完成
- RBAC服务初始化完成
- API处理器初始化完成
- 路由注册完成
- HTTP服务器创建完成
- 服务器启动

### 7.2 刮削相关

- 刮削系统启动/停止
- 工作协程启动
- 任务提交
- 任务开始处理
- 任务处理完成
- 刮削结果存储

### 7.3 API请求

- 请求方法、路径、状态码
- 请求处理时间
- 客户端IP

## 8. 使用示例

### 8.1 基本日志记录

```go
logger.Info("服务器启动中...")
logger.Error("服务器启动失败: %v", err)
logger.Debug("调试信息: %v", value)
logger.Warn("警告信息: %v", value)
```

### 8.2 结构化日志

```go
logger.InfoJSON(map[string]interface{}{
    "task_id": task.ID.Hex(),
    "module":  task.Module,
    "status":  "success",
})
```

## 9. 时间格式

日志时间使用 RFC3339 格式：

```
2024-01-15T10:30:00Z
```

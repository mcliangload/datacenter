# 数据中心系统代码结构

## 1. 目录结构

```
datacenter/
├── cmd/
│   └── server/           # 应用入口
│       └── main.go       # 主程序入口
├── internal/
│   ├── api/              # HTTP处理器
│   │   ├── handlers.go   # 业务逻辑处理器
│   │   ├── middleware/   # 中间件
│   │   │   ├── auth.go   # 认证中间件
│   │   │   └── logger.go # 日志中间件
│   │   └── routes.go     # 路由配置
│   ├── auth/             # 认证相关
│   │   ├── jwt.go        # JWT实现
│   │   └── test/         # 测试
│   ├── logger/           # 日志系统
│   │   └── logger.go     # 日志实现
│   ├── models/           # 数据模型
│   │   └── models.go     # 模型定义
│   ├── scraper/          # 刮削系统
│   │   ├── scraper.go    # 刮削器实现
│   │   ├── task.go       # 任务管理
│   │   └── worker.go     # 工作协程
│   └── storage/          # 数据存储层
│       ├── mongodb.go     # MongoDB存储实现
│       └── rbac_storage.go # RBAC存储实现
├── pkg/
│   ├── rbac/             # RBAC服务
│   │   ├── rbac.go       # RBAC核心实现
│   │   └── test/         # 测试
│   └── jql/              # JQL查询
│       └── parser.go      # JQL解析器
├── docs/                 # 文档
│   ├── architecture.md   # 架构设计
│   ├── business-data.md  # 业务数据管理
│   ├── rbac.md           # RBAC系统
│   ├── logging.md        # 日志系统
│   └── query.md          # 查询功能
├── config/               # 配置文件
│   └── config.go         # 配置加载
├── scripts/              # 脚本
│   └── build.sh          # 构建脚本
├── go.mod                # Go模块定义
└── go.sum                # 依赖校验
```

## 2. 模块职责说明

### 2.1 核心模块

| 模块 | 主要职责 | 文件位置 | 依赖关系 |
|------|----------|----------|----------|
| **main** | 应用入口，初始化服务 | cmd/server/main.go | 所有模块 |
| **API** | 处理HTTP请求，路由管理 | internal/api/ | auth, logger, models, storage, scraper |
| **Auth** | JWT认证，密码加密 | internal/auth/ | logger |
| **Logger** | 日志记录，中间件 | internal/logger/ | - |
| **Models** | 数据模型定义 | internal/models/ | - |
| **Scraper** | 异步刮削任务处理 | internal/scraper/ | logger, storage |
| **Storage** | MongoDB数据访问 | internal/storage/ | models |
| **RBAC** | 权限管理服务 | pkg/rbac/ | storage |
| **JQL** | 查询语句解析 | pkg/jql/ | - |

### 2.2 新增模块

| 模块 | 主要职责 | 文件位置 | 依赖关系 |
|------|----------|----------|----------|
| **Scraper** | 异步处理刮削任务，任务队列，并发执行 | internal/scraper/ | logger, storage |
| **Scraper Task** | 刮削任务管理，状态更新 | internal/scraper/task.go | storage |
| **Scraper Worker** | 工作协程池，任务执行 | internal/scraper/worker.go | logger |

## 3. 关键文件说明

### 3.1 入口文件

- **cmd/server/main.go**：应用程序入口，负责初始化服务、连接数据库、启动HTTP服务器

### 3.2 API层

- **internal/api/routes.go**：定义所有API路由，包括认证、业务数据、刮削任务等
- **internal/api/handlers.go**：实现HTTP请求处理逻辑
- **internal/api/middleware/auth.go**：JWT认证中间件
- **internal/api/middleware/logger.go**：HTTP请求日志中间件

### 3.3 业务逻辑层

- **internal/scraper/scraper.go**：刮削系统核心，管理任务队列和工作协程
- **internal/scraper/task.go**：刮削任务管理，包括创建、更新、查询任务
- **internal/scraper/worker.go**：工作协程实现，执行刮削器脚本

### 3.4 数据访问层

- **internal/storage/mongodb.go**：MongoDB存储实现，支持动态集合管理
- **internal/storage/rbac_storage.go**：RBAC专用存储实现

### 3.5 模型层

- **internal/models/models.go**：定义所有数据模型，包括用户、权限、角色、刮削任务等

### 3.6 服务层

- **pkg/rbac/rbac.go**：RBAC服务实现，提供权限检查和管理功能
- **pkg/jql/parser.go**：JQL查询解析器

## 4. 关键流程

### 4.1 刮削任务处理流程

1. **任务创建**：API接收刮削任务请求，验证参数，创建任务记录
2. **任务提交**：将任务提交到后台队列
3. **异步处理**：工作协程从队列中获取任务并执行
4. **状态更新**：执行完成后更新任务状态和结果
5. **日志记录**：记录任务执行的详细日志

### 4.2 权限检查流程

1. **请求验证**：认证中间件验证JWT Token
2. **权限检查**：RBAC服务检查用户权限
3. **授权决策**：根据权限检查结果允许或拒绝请求
4. **日志记录**：记录权限检查和授权决策

### 4.3 动态集合管理流程

1. **集合创建**：首次上传数据时自动创建集合
2. **索引配置**：自动配置集合索引
3. **角色创建**：为集合创建对应的RBAC角色
4. **权限分配**：将角色分配给datatypeowner

## 5. 配置管理

系统使用环境变量和配置文件进行配置管理：

| 配置项 | 描述 | 默认值 |
|--------|------|--------|
| MONGODB_URI | 业务数据库连接URI | - |
| MONGODB_DATABASE | 业务数据库名称 | - |
| MONGODB_RBAC_URI | RBAC数据库连接URI | - |
| MONGODB_RBAC_DATABASE | RBAC数据库名称 | - |
| JWT_SECRET | JWT签名密钥 | - |
| JWT_EXPIRATION | JWT过期时间（小时） | 24 |
| LOG_LEVEL | 日志级别 | info |
| LOG_HTTP_FILE | HTTP日志文件路径 | logs/http.log |
| LOG_APP_FILE | 应用日志文件路径 | logs/app.log |
| LOG_SCRAPER_FILE | 刮削日志文件路径 | logs/scraper.log |
| LOG_AUDIT_FILE | 审计日志文件路径 | logs/audit.log |
| SCRAPER_WORKERS | 刮削工作协程数量 | 4 |
| SCRAPER_QUEUE_SIZE | 刮削任务队列大小 | 1000 |

## 6. 日志管理

系统使用分层日志管理：

- **HTTP日志**：记录所有API请求和响应
- **应用日志**：记录应用程序运行时信息
- **刮削日志**：记录刮削任务执行情况
- **审计日志**：记录权限变更和重要操作

## 7. 安全措施

- **密码加密**：使用bcrypt加密存储用户密码
- **JWT认证**：基于Token的无状态认证
- **权限控制**：基于RBAC模型的细粒度权限管理
- **数据库隔离**：业务数据和RBAC数据使用独立数据库
- **输入验证**：对所有用户输入进行验证
- **日志审计**：记录所有重要操作和权限变更

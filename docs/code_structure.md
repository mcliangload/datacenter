# 数据中心系统代码结构

## 1. 目录结构

```
datacenter/
├── cmd/
│   └── server/
│       └── main.go              # 应用入口，初始化服务、启动服务器
├── internal/
│   ├── api/
│   │   └── handlers.go          # HTTP处理器，所有RESTful接口实现
│   ├── auth/
│   │   ├── jwt.go              # JWT服务，Token生成和验证
│   │   ├── middleware.go        # 认证中间件和权限中间件
│   │   └── test/
│   │       └── jwt_test.go     # JWT服务测试
│   ├── logger/
│   │   ├── logger.go           # 日志初始化，zerolog + lumberjack
│   │   └── middleware.go       # HTTP日志中间件
│   ├── models/
│   │   └── models.go           # 数据模型定义，含字段验证逻辑
│   ├── scraper/
│   │   └── scraper.go          # 刮削系统，工作队列和Worker池
│   └── storage/
│       ├── mongodb.go          # MongoDB连接管理
│       ├── mongodb_storage.go  # 业务数据存储实现
│       └── rbac_storage.go     # RBAC存储实现
├── pkg/
│   ├── rbac/
│   │   └── rbac.go             # RBAC权限服务，通配符匹配
│   └── jql/
│       ├── parser.go           # JQL查询解析器
│       └── parser_test.go      # JQL解析器测试
├── logs/                        # 日志文件目录
├── docs/                        # 文档目录
├── go.mod                       # Go模块定义
└── go.sum                       # 依赖校验
```

## 2. 模块职责说明

### 2.1 核心模块

| 模块 | 主要职责 | 文件位置 | 依赖关系 |
|------|----------|----------|----------|
| **main** | 应用入口，初始化服务 | cmd/server/main.go | 所有模块 |
| **API** | 处理HTTP请求，路由管理 | internal/api/ | auth, logger, models, storage, scraper |
| **Auth** | JWT认证，密码加密 | internal/auth/ | logger |
| **Logger** | 日志记录，中间件 | internal/logger/ | - |
| **Models** | 数据模型定义，字段验证 | internal/models/ | - |
| **Scraper** | 异步刮削任务处理 | internal/scraper/ | logger, storage |
| **Storage** | MongoDB数据访问 | internal/storage/ | models |
| **RBAC** | 权限管理服务 | pkg/rbac/ | storage |
| **JQL** | 查询语句解析 | pkg/jql/ | - |

## 3. 关键文件说明

### 3.1 入口文件

- **cmd/server/main.go**：应用程序入口，负责：
  - 加载环境变量（.env文件）
  - 初始化日志系统
  - 初始化业务存储（MongoDB datacenter数据库）
  - 初始化RBAC存储（MongoDB rbac数据库）
  - 初始化默认数据
  - 启动刮削系统
  - 初始化JWT服务
  - 初始化RBAC服务
  - 创建并启动HTTP服务器
  - 优雅关闭处理

### 3.2 API层

- **internal/api/handlers.go**：实现所有HTTP请求处理逻辑，包括：
  - 认证接口：登录、注册
  - 用户管理CRUD
  - 权限管理CRUD
  - 角色管理CRUD
  - 字段定义管理CRUD
  - 业务数据管理CRUD
  - 集合管理CRUD
  - 刮削任务管理
  - 已删除数据管理

### 3.3 认证层

- **internal/auth/jwt.go**：JWT服务实现
  - Token生成（GenerateToken）
  - Token验证（ValidateToken）
  - 密码哈希（HashPassword）
  - 密码验证（CheckPassword）

- **internal/auth/middleware.go**：中间件实现
  - AuthMiddleware：JWT认证中间件
  - PermissionMiddleware：权限检查中间件，支持通配符匹配

### 3.4 日志层

- **internal/logger/logger.go**：日志初始化
  - 使用zerolog实现结构化日志
  - 使用lumberjack实现日志轮转
  - 应用日志同时输出到stdout和文件

- **internal/logger/middleware.go**：HTTP日志中间件
  - 记录请求方法、路径、状态码、延迟时间、客户端IP

### 3.5 数据模型层

- **internal/models/models.go**：定义所有数据模型
  - BaseModel：基础模型（CreatedBy, CreatedAt, UpdatedBy, UpdatedAt）
  - User：用户模型
  - Permission：权限模型
  - Role：角色模型
  - FieldDefinition：字段定义模型，含验证逻辑
  - Constraints：字段约束（min, max, min_length, max_length, pattern, enum_values）
  - BusinessData：业务数据模型
  - DeletedData：已删除业务数据模型
  - ScrapeTask：刮削任务模型
  - DeletedScrapeTask：已删除刮削任务模型
  - Collection：集合元数据模型

### 3.6 刮削系统

- **internal/scraper/scraper.go**：刮削系统核心实现
  - Scraper接口：SubmitTask, Start, Stop
  - 工作队列（Channel）管理
  - Worker协程池
  - 任务处理流程
  - 刮削器执行
  - 结果存储

### 3.7 存储层

- **internal/storage/mongodb.go**：MongoDB连接管理
  - NewMongoDBStorage：创建业务存储实例
  - NewRBACMongoDBStorage：创建RBAC存储实例

- **internal/storage/mongodb_storage.go**：业务数据存储实现
  - 业务数据CRUD
  - 字段定义CRUD
  - 集合管理
  - 刮削任务CRUD
  - 软删除和恢复

- **internal/storage/rbac_storage.go**：RBAC存储实现
  - 用户CRUD
  - 权限CRUD
  - 角色CRUD
  - 默认数据初始化

### 3.8 服务层

- **pkg/rbac/rbac.go**：RBAC权限服务
  - CheckPermission：权限检查
  - GetUserPermissions：获取用户权限
  - 通配符权限匹配（如 user:* 匹配 user:read）

- **pkg/jql/parser.go**：JQL查询解析器
  - ParseQuery：解析JQL查询语句为MongoDB查询filter

## 4. 关键流程

### 4.1 系统启动流程

```
1. 加载环境变量 (godotenv.Load)
2. 初始化日志 (logger.Init)
3. 初始化业务存储 (storage.NewMongoDBStorage)
4. 初始化RBAC存储 (storage.NewRBACMongoDBStorage)
5. 初始化默认数据 (rbacStorage.InitDefaultData)
6. 启动刮削系统 (scraper.Start)
7. 初始化JWT服务 (auth.NewJWTService)
8. 初始化RBAC服务 (rbac.NewService)
9. 创建API处理器 (api.NewHandler)
10. 注册路由 (handler.RegisterRoutes)
11. 启动HTTP服务器
```

### 4.2 刮削任务处理流程

```
1. 用户提交任务: POST /api/scraper/upload
2. 创建任务记录: 状态为pending，保存到scrape_tasks集合
3. 任务入队: 放入Channel队列
4. Worker取出任务: 更新状态为scraping
5. 执行刮削器: python {scraper_path} {data_path}
6. 解析结果:
   - 成功: 状态改为success，存储结果到{module}_data集合
   - 失败: 状态改为failed，记录错误信息
7. 更新任务记录: 更新business_data_id关联
```

### 4.3 权限检查流程

```
1. 用户请求携带JWT Token
2. AuthMiddleware验证Token，解析user_id和roles
3. 获取用户权限: rbacService.GetUserPermissions
4. PermissionMiddleware检查所需权限
5. 支持通配符匹配: 如user:*可匹配user:read, user:write
6. 有权限放行，无权限返回403
```

### 4.4 字段验证流程

```
1. 定义字段约束: FieldDefinition包含Constraints
2. 提交数据时: 根据field_definitions验证数据
3. 支持的约束类型:
   - string: min_length, max_length, pattern, enum_values
   - number: min, max
   - array: list_min_length, list_max_length
4. 验证失败返回错误信息和具体字段
```

## 5. 配置管理

系统使用环境变量和配置文件进行配置管理：

| 配置项 | 描述 | 默认值 |
|--------|------|--------|
| SERVER_HOST | 服务监听地址 | 0.0.0.0 |
| SERVER_PORT | 服务监听端口 | 8080 |
| MONGODB_URI | 业务数据库连接URI | mongodb://localhost:27017 |
| MONGODB_DATABASE | 业务数据库名称 | datacenter |
| MONGODB_RBAC_URI | RBAC数据库连接URI | mongodb://localhost:27017 |
| MONGODB_RBAC_DATABASE | RBAC数据库名称 | rbac |
| JWT_SECRET | JWT签名密钥 | your-secret-key |
| JWT_EXPIRATION | JWT过期时间（小时） | 24 |
| JWT_REFRESH_EXPIRATION | 刷新Token过期时间（小时） | 720 |
| LOG_LEVEL | 日志级别 | info |
| LOG_HTTP_FILE | HTTP日志文件路径 | logs/http.log |
| LOG_MAX_SIZE | 日志文件最大大小(MB) | 100 |
| LOG_MAX_BACKUPS | 日志备份数 | 5 |
| LOG_MAX_AGE | 日志保留天数 | 30 |
| SCRAPER_WORKERS | 刮削工作协程数 | 4 |

## 6. 日志管理

系统使用分层日志管理：

- **HTTP日志**：logs/http.log，记录所有API请求和响应
- **应用日志**：logs/app.log，记录程序运行时信息

日志格式（JSON）：
```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "caller": "handler.go:123",
  "message": "Request processed"
}
```

## 7. 安全措施

- **密码加密**：使用bcrypt加密存储用户密码
- **JWT认证**：基于Token的无状态认证
- **权限控制**：基于RBAC模型的细粒度权限管理，支持通配符匹配
- **数据库隔离**：业务数据和RBAC数据使用独立数据库
- **输入验证**：对所有用户输入进行验证
- **CORS**：支持跨域配置

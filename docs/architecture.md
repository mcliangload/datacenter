# 数据中心系统架构设计文档

## 1. 系统总体架构概述

本数据中心系统采用前后端分离的分层架构设计。后端基于 Go 语言、Gin 框架和 MongoDB 构建，前端基于 React + TypeScript + Ant Design。系统提供企业级数据管理、用户权限控制（双层 RBAC）、异步数据刮削、自定义字段验证、JQL 高级查询和审计日志等核心功能。

### 1.1 架构分层

```
┌─────────────────────────────────────────────────────────────────┐
│                    客户端层 (Client Layer)                        │
│                 React 18 + TypeScript 5 + Ant Design 5            │
│                  Vite 5 构建 / React Router v6 路由                │
└─────────────────────────────────────────────────────────────────┘
                              │ HTTPS
┌─────────────────────────────────────────────────────────────────┐
│                      API 层 (API Layer)                           │
│                  Gin Router + 中间件链 + Handlers                  │
│  ┌──────────┬──────────┬───────────┬───────────┬──────────┐    │
│  │ Auth API │ User API │ RBAC API  │Business API│Scrape API│    │
│  └──────────┴──────────┴───────────┴───────────┴──────────┘    │
├─────────────────────────────────────────────────────────────────┤
│                     服务层 (Service Layer)                        │
│  ┌──────────┬──────────┬───────────┬───────────┬──────────┐    │
│  │JWT Service│RBAC Svc │Coll RBAC  │JQL Parser │ Scraper  │    │
│  │  (auth)   │  (rbac) │   (rbac)  │  (jql)    │(scraper) │    │
│  └──────────┴──────────┴───────────┴───────────┴──────────┘    │
├─────────────────────────────────────────────────────────────────┤
│                    存储层 (Storage Layer)                         │
│  ┌──────────────────────────────────┐ ┌────────────────────────┐ │
│  │     业务数据库 (datacenter)        │ │    RBAC 数据库 (rbac)    │ │
│  │  collections / field_definitions │ │  users / permissions   │ │
│  │  scrape_tasks / {module}_data    │ │  roles / collection_*  │ │
│  │  deleted_data / audit_logs       │ │  audit_logs            │ │
│  └──────────────────────────────────┘ └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 核心功能模块

| 模块 | 描述 | 关键实现 |
|------|------|----------|
| 认证授权 | 基于 JWT Token 的认证机制，支持 Token 刷新和过期策略 | `internal/auth/jwt.go` |
| 系统级 RBAC | 用户-角色-权限三级模型，通配符匹配，超级管理员 | `pkg/rbac/rbac.go` |
| 集合级 RBAC | 模块级权限控制，Owner/Operator/Tourist 三级角色 | `pkg/rbac/collection_rbac.go` |
| 业务数据管理 | 动态集合 CRUD，自定义字段验证，软删除和恢复 | `internal/api/handlers.go` |
| 数据刮削系统 | 协程池 + Channel 队列异步处理，状态跟踪，重试机制 | `internal/scraper/scraper.go` |
| JQL 查询引擎 | 自定义查询语法解析为 MongoDB 过滤器 | `pkg/jql/parser.go` |
| 集合管理 | 动态 MongoDB 集合创建/删除，索引管理 | `internal/storage/` |
| 审计日志 | 集合级操作审计追踪 | `internal/models/models.go` |

## 2. 技术栈

| 分类 | 技术/框架 | 版本 | 说明 |
|------|----------|------|------|
| **后端** ||||
| 语言 | Go | 1.20+ | 编译型语言，高并发支持 |
| Web 框架 | Gin | v1.9.1 | 轻量级高性能 HTTP 框架 |
| 数据库 | MongoDB | 6.0+ | 文档型数据库，支持动态 Schema |
| 驱动 | mongo-driver | v1.12.1 | 官方 Go 驱动 |
| 认证 | golang-jwt/jwt | v5.0.0 | JWT 令牌管理 |
| 密码 | bcrypt | v0.14.0 | 安全的密码哈希 |
| 日志 | zerolog + lumberjack | v1.31.0 / v2.2.1 | 结构化日志 + 自动轮转 |
| UUID | google/uuid | v1.6.0 | 唯一标识符生成 |
| **前端** ||||
| 框架 | React | 18.2+ | 声明式 UI 组件 |
| 语言 | TypeScript | 5.2+ | 类型安全 |
| UI 库 | Ant Design | 5.12+ | 企业级 UI 组件库 |
| 构建 | Vite | 5+ | 快速开发与构建工具 |
| 路由 | React Router | v6.21+ | 客户端路由管理 |
| 状态管理 | Zustand | 4.4+ | 轻量级状态管理 |
| HTTP 客户端 | Axios | 1.6+ | HTTP 请求封装 |

## 3. 项目目录结构与模块职责

### 3.1 目录结构

```
datacenter/
├── cmd/
│   └── server/
│       └── main.go                    # 应用入口，依赖注入与启动编排
├── cmdenscrape/
│   └── main.go                        # 刮削测试数据生成器
├── configs/
│   └── config.yaml                    # YAML 配置文件
├── internal/
│   ├── api/
│   │   ├── handlers.go                # 全部 RESTful Handler (路由注册与业务逻辑)
│   │   └── collection_permission_middleware.go  # 集合权限中间件
│   ├── auth/
│   │   ├── jwt.go                     # JWT 服务（生成/验证/刷新）
│   │   ├── middleware.go              # 认证与权限中间件
│   │   └── test/
│   │       └── jwt_test.go            # JWT 单元测试
│   ├── logger/
│   │   ├── logger.go                  # 日志初始化（zerolog + lumberjack）
│   │   └── middleware.go              # HTTP 请求日志中间件
│   ├── models/
│   │   └── models.go                  # 全部数据模型定义及字段验证逻辑
│   ├── scraper/
│   │   └── scraper.go                 # 刮削任务协程池系统
│   └── storage/
│       ├── mongodb.go                 # MongoDB 连接管理
│       ├── mongodb_storage.go         # 业务数据存储层
│       ├── rbac_storage.go            # RBAC 数据存储层
│       └── collection_rbac_storage.go # 集合 RBAC 存储层
├── pkg/
│   ├── jql/
│   │   ├── parser.go                  # JQL 查询语言词法/语法解析器
│   │   └── parser_test.go             # 解析器单元测试
│   └── rbac/
│       ├── rbac.go                    # 系统级 RBAC 权限服务
│       └── collection_rbac.go         # 集合级 RBAC 权限服务
├── frontend/
│   ├── src/
│   │   ├── App.tsx                    # 前端主路由与布局
│   │   ├── main.tsx                   # Vite 入口
│   │   ├── pages/                     # 页面组件
│   │   ├── services/                  # API 调用服务层
│   │   ├── stores/                    # Zustand 状态管理
│   │   ├── theme/                     # Ant Design 主题配置
│   │   └── types/                     # TypeScript 类型定义
│   ├── dist/                          # 构建产物
│   └── package.json                   # 前端依赖
├── docs/                              # 项目文档
├── logs/                              # 运行时日志目录
├── .env                               # 环境变量配置
└── go.mod                             # Go 模块定义
```

### 3.2 模块职责

| 包 | 路径 | 职责 |
|------|------|------|
| API Handlers | `internal/api` | 处理所有 HTTP 请求，实现 RESTful 接口，注册路由与中间件 |
| Auth | `internal/auth` | JWT Token 生成/验证/刷新、bcrypt 密码加密/校验 |
| Logger | `internal/logger` | zerolog 日志初始化、HTTP 请求/响应日志中间件 |
| Models | `internal/models` | 定义 User、Role、Permission、BusinessData、ScrapeTask、CollectionRole 等全部数据结构及字段验证方法 |
| Storage | `internal/storage` | MongoDB 数据访问层，业务数据库与 RBAC 数据库物理隔离 |
| Scraper | `internal/scraper` | 协程池 + Channel 任务队列架构，异步执行刮削器脚本 |
| JQL Parser | `pkg/jql` | JQL 查询语言的手写递归下降解析器，将查询字符串转换为 MongoDB bson.M |
| RBAC | `pkg/rbac` | 全局 RBAC 权限检查（通配符匹配）、集合级 RBAC（Owner/Operator/Tourist） |
| Collection Permission | `internal/api` | 集合级权限中间件工厂函数，在路由层拦截鉴权 |

## 4. 应用启动流程

应用入口位于 `cmd/server/main.go`，采用显式的依赖注入模式进行初始化编排：

```
1.  加载 .env 环境变量 (godotenv)
2.  初始化日志系统 (zerolog + lumberjack)
3.  初始化业务数据库连接 (MongoDB datacenter)
4.  初始化 RBAC 数据库连接 (MongoDB rbac)
5.  初始化默认 RBAC 数据 (admin 用户、默认角色和权限)
6.  启动刮削系统 (协程池 worker pool)
7.  初始化 JWT 服务
8.  初始化系统 RBAC 服务
9.  初始化集合 RBAC 存储与服务
10. 创建 API Handler (注入全部依赖)
11. 注册路由 (Gin Router)
12. 启动 HTTP 服务器并监听优雅关闭信号
```

**核心设计模式**: Handler 构造函数 `NewHandler()` 接收所有依赖（存储层、刮削系统、JWT 服务、两种 RBAC 服务），通过接口而非具体实现进行解耦。

## 5. 数据库架构

### 5.1 双数据库设计

系统使用两个独立的 MongoDB 数据库实现数据与权限的物理隔离：

```
┌─────────────────────────────────────────────────────────────┐
│                     MongoDB 集群                             │
├─────────────────────────┬─────────────────────────────────┤
│    datacenter 数据库      │         rbac 数据库              │
├─────────────────────────┼─────────────────────────────────┤
│ collections             │ users                            │
│ field_definitions       │ permissions                      │
│ scrape_tasks            │ roles                            │
│ deleted_scrape_tasks    │ collection_roles                 │
│ {module}_data (动态)    │ collection_role_assignments      │
│ deleted_data            │ audit_logs                       │
└─────────────────────────┴─────────────────────────────────┘
```

### 5.2 业务数据库集合 (datacenter)

| 集合名 | 说明 | 动态创建 |
|--------|------|----------|
| `collections` | 模块（集合）元数据，含 module、datatype_owner 等 | 否 |
| `field_definitions` | 字段定义，含字段类型和验证约束 | 否 |
| `scrape_tasks` | 刮削任务（pending/scraping/success/failed） | 否 |
| `deleted_scrape_tasks` | 软删除的刮削任务，含 original_id | 否 |
| `{module}_data` | 模块业务数据（动态集合名，如 movie_data） | 是 |
| `deleted_data` | 软删除的业务数据，含 original_id 和 deleted_at | 否 |

### 5.3 RBAC 数据库集合 (rbac)

| 集合名 | 说明 |
|--------|------|
| `users` | 用户信息，含 bcrypt 加密密码和 role_ids 数组 |
| `permissions` | 权限定义，code 为唯一键 |
| `roles` | 角色定义，含 permission_ids 数组 |
| `collection_roles` | 集合级角色定义（Owner/Operator/Tourist），含 permission 代码数组 |
| `collection_role_assignments` | 用户到集合角色的分配关系 |
| `audit_logs` | 集合操作审计日志 |

## 6. 数据模型设计

### 6.1 BaseModel（公共基类）

所有业务模型组合该结构体，统一审计字段：

```go
type BaseModel struct {
    CreatedBy string    `json:"created_by" bson:"created_by"`
    CreatedAt time.Time `json:"created_at" bson:"created_at"`
    UpdatedBy string    `json:"updated_by" bson:"updated_by"`
    UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}
```

### 6.2 核心数据模型一览

| 模型 | 所属数据库 | 说明 |
|------|-----------|------|
| `User` | rbac | 用户，含 username/email/password/role_ids |
| `Permission` | rbac | 权限定义，含 name/code |
| `Role` | rbac | 角色，含 name/code/permission_ids |
| `FieldDefinition` | datacenter | 字段定义，含 field_type/constraints 验证规则 |
| `BusinessData` | datacenter | 业务数据，含 module/description/custom_fields/file_path |
| `DeletedData` | datacenter | 软删除数据，含 original_id/deleted_at |
| `ScrapeTask` | datacenter | 刮削任务，含 status/result/error_message/business_data_id |
| `DeletedScrapeTask` | datacenter | 软删除的刮削任务 |
| `Collection` | datacenter | 集合元数据，含 module/datatype_owner/collection_name |
| `CollectionRole` | rbac | 集合角色，含 type（owner/operator/tourist）和 permission 代码 |
| `CollectionRoleAssignment` | rbac | 用户-集合角色分配关系 |
| `AuditLog` | rbac | 审计日志，含 user_id/action/resource/details/ip_address |

### 6.3 字段类型与验证约束

系统支持 6 种字段类型，通过 `FieldDefinition.Validate()` 方法进行运行时验证：

| 字段类型 | 支持的约束 |
|----------|-----------|
| `string` | min_length, max_length, pattern (正则), enum_values |
| `number` | min, max |
| `boolean` | 类型校验 |
| `date` | RFC3339 格式校验 |
| `array` | list_min_length, list_max_length |
| `object` | 基本校验 |

## 7. API 架构

### 7.1 中间件链

请求处理经过以下中间件链（按顺序）：

```
CORS 中间件 → Logger 中间件 → Gin Recovery → 路由匹配
    ├── /api/auth/* → 无需认证（公开）
    └── /api/* → AuthMiddleware (JWT验证)
                    ├── PermissionMiddleware (全局权限)
                    └── CollectionPermissionMiddleware (集合级权限)
```

### 7.2 API 路由全览

```
/api
├── public (无需认证)
│   ├── POST /auth/login              # 用户登录，返回 JWT Token
│   └── POST /auth/register           # 用户注册
│
└── protected (需要 JWT + 权限)
    │
    ├── /users                        # 用户管理 (user:read / user:write)
    │   ├── GET    ""                 # 用户列表（分页）
    │   ├── GET    /:id               # 用户详情
    │   ├── POST   ""                 # 创建用户
    │   ├── PUT    /:id               # 更新用户
    │   ├── DELETE /:id               # 删除用户
    │   ├── POST   /:id/roles         # 分配角色
    │   ├── DELETE /:id/roles/:roleId # 移除角色
    │   └── GET    /:id/roles         # 获取用户角色
    │
    ├── /permissions                  # 权限管理 (permission:read / permission:write)
    │   └── CRUD: GET/POST/PUT/DELETE
    │
    ├── /roles                        # 角色管理 (role:read / role:write)
    │   ├── CRUD: GET/POST/PUT/DELETE
    │   ├── POST   /:id/permissions       # 分配权限
    │   ├── DELETE /:id/permissions/:pid  # 移除权限
    │   └── GET    /:id/permissions       # 获取角色权限
    │
    ├── /fields                        # 字段定义 (双重权限：全局 + 集合级)
    │   ├── GET    /module/:module    # 按模块获取（需集合读权限）
    │   ├── GET    /:id               # 获取详情（需 field:read）
    │   ├── POST   ""                 # 创建（需集合 field:admin）
    │   ├── PUT    /:id               # 更新（需集合 field:admin）
    │   └── DELETE /:id               # 删除（需集合 field:admin）
    │
    ├── /business 或 /collection-data/module/:module
    │   ├── POST   ""                   # 创建业务数据（需集合写权限）
    │   ├── GET    /module/:module      # 分页查询（支持 JQL，需集合读权限）
    │   ├── GET    /module/:module/:id  # 获取详情
    │   ├── PUT    /module/:module/:id  # 更新
    │   └── DELETE /module/:module/:id  # 软删除（需集合删权限）
    │
    ├── /deleted                       # 已删除数据 (data:read / data:write)
    │   ├── GET  /module/:module       # 按模块查看
    │   ├── GET  /:id                  # 查看详情
    │   └── POST /:id/recover          # 恢复数据
    │
    ├── /scraper                       # 刮削任务 (scrape:read / scrape:write)
    │   ├── POST   /upload             # 提交刮削任务
    │   ├── GET    /tasks              # 任务列表（分页，按 module/status 过滤）
    │   ├── GET    /tasks/:id          # 任务详情
    │   ├── POST   /tasks/:id/retry    # 重试失败任务
    │   ├── DELETE /tasks/:id          # 软删除单个任务
    │   └── POST   /tasks/batch-delete # 批量软删除
    │
    ├── /deleted-scraper               # 已删除刮削任务
    │   ├── GET  /module/:module       # 按模块查看
    │   ├── GET  /:id                  # 查看详情
    │   └── POST /:id/recover          # 恢复任务
    │
    └── /collections                   # 集合管理 (collection:read / collection:write)
        ├── GET    ""                           # 集合列表
        ├── GET    /:module                     # 集合详情
        ├── POST   ""                           # 创建集合（自动创建 RBAC 角色）
        ├── PUT    /:module                     # 更新集合（自动转移 Owner）
        ├── DELETE /:module                     # 级联删除（集合+数据+角色+权限）
        ├── POST   /:module/indexes             # 创建索引
        ├── GET    /:module/indexes             # 获取索引列表
        ├── DELETE /:module/indexes/:name       # 删除索引
        ├── GET    /:module/roles               # 获取集合角色
        ├── GET    /:module/roles/assignments   # 获取角色分配
        ├── POST   /:module/roles/assign        # 分配集合角色
        └── DELETE /:module/roles/:rid/assignments/:uid  # 移除集合角色
```

### 7.3 分页响应格式

```json
{
    "data": [...],
    "total": 100,
    "page": 1,
    "pageSize": 10
}
```

## 8. 认证与授权架构

### 8.1 JWT 认证流程

```
┌─────────┐     POST /api/auth/login      ┌──────────┐
│  客户端   │ ─────────────────────────────▶ │  服务器   │
│         │                                │          │
│         │ ◀───────────────────────────── │          │
│         │   { token, user }              │  验证凭证  │
│         │                                │  bcrypt   │
│         │    Authorization: Bearer <JWT>  │  比对    │
│         │ ─────────────────────────────▶ │          │
│         │                                │ 查询角色  │
│         │                                │ 与权限    │
│         │                                │ 签发 JWT  │
└─────────┘                                └──────────┘
```

**JWT Claims 结构**:
```go
type Claims struct {
    UserID      string   `json:"user_id"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
    jwt.RegisteredClaims  // Issuer: "datacenter", 含 exp/iat/nbf/sub/jti
}
```

- **Token 有效期**: 默认 24 小时 (`JWT_EXPIRATION`)
- **刷新窗口**: Token 过期后 720 小时内可刷新 (`JWT_REFRESH_EXPIRATION`)
- **签名算法**: HMAC-SHA256
- **密码安全**: bcrypt（DefaultCost = 10）

### 8.2 系统级 RBAC 模型

```
┌─────────┐  N:M   ┌─────────┐  N:M   ┌────────────┐
│   User   │───────▶│   Role   │───────▶│ Permission  │
├─────────┤        ├─────────┤        ├────────────┤
│ role_ids│        │perm_ids │        │ code        │
└─────────┘        └─────────┘        └────────────┘
```

**权限代码体系** (资源:操作):

| 权限代码 | 描述 |
|----------|------|
| `user:read` / `user:write` / `user:manage` | 用户管理 |
| `role:read` / `role:write` / `role:manage` | 角色管理 |
| `permission:read` / `permission:write` / `permission:manage` | 权限管理 |
| `data:read` / `data:write` / `data:manage` | 数据管理 |
| `field:read` / `field:write` / `field:manage` | 字段管理 |
| `scrape:read` / `scrape:write` / `scrape:manage` | 刮削管理 |
| `collection:read` / `collection:write` / `collection:manage` | 集合管理 |
| `system:admin` | 超级管理员（绕过所有权限检查） |

**通配符匹配规则**: 代码 `user:*` 可匹配 `user:read`、`user:write` 等所有 user 前缀权限。

### 8.3 集合级 RBAC 模型

集合级 RBAC 在系统级 RBAC 之上提供模块（collection）粒度的权限控制：

```
┌──────────────┐     ┌───────────────────────────┐
│    User       │────▶│ CollectionRoleAssignment   │
│   (rbac)     │     │  user_id + module + role_id │
└──────────────┘     └───────────────────────────┘
                                    │
┌───────────────────────────────────▼──────────────────────────────┐
│                    CollectionRole (rbac)                         │
│  Type: owner | operator | tourist                                │
│  PermissionIDs: ["movie:read", "movie:write", ...]               │
└──────────────────────────────────────────────────────────────────┘
```

**三级集合角色**:

| 角色类型 | 常量 | 权限 |
|----------|------|------|
| Owner（集合管理员） | `CollectionRoleTypeOwner` | admin, read, write, delete, field:admin（全部权限） |
| Operator（数据操作员） | `CollectionRoleTypeOperator` | read, write, delete（数据增删改查，不能修改字段定义） |
| Tourist（普通用户） | `CollectionRoleTypeTourist` | read（仅查看） |

**集合权限代码**:
- `{module}:read` — 读取集合数据
- `{module}:write` — 写入集合数据
- `{module}:delete` — 删除集合数据
- `{module}:admin` — 管理集合（含角色分配）
- `{module}:field:admin` — 管理自定义字段定义

**创建集合时的自动操作**:
1. 在 `rbac.permissions` 创建 5 个模块权限
2. 在 `rbac.roles` 创建 3 个系统角色（Owner/Operator/Tourist）
3. 在 `rbac.collection_roles` 创建 3 个集合角色
4. 将 Owner 角色自动分配给 `datatype_owner` 指定的用户

**权限检查流程**:
1. 先检查用户是否拥有 `system:admin`（超级管理员绕过）
2. 检查系统级 Role 中是否包含集合权限（如 `movie:read`）
3. 检查用户在 `collection_role_assignments` 中是否有该模块的集合角色分配
4. 如果有，检查该集合角色是否包含所需权限

### 8.4 中间件权限控制模式

路由注册时采用两种中间件模式：

**全局权限中间件** (`PermissionMiddleware`): 检查用户系统级角色中的权限 code。

**集合权限中间件** (`CollectionPermissionMiddleware`): 从 URL 参数 `:module` 或请求体 `module` 字段获取模块名，然后检查集合级权限。

```
路由注册示例：
┌──────────────────────────────────────────────────────────────┐
│ // 全局权限：创建用户需要 user:write                          │
│ users.POST("", h.CreateUser)                                 │
│   .Use(h.PermissionMiddleware(rbac.PermissionUserWrite))     │
│                                                              │
│ // 集合权限：读取 movie 数据需要 movie:read                   │
│ business.GET("/module/:module",                              │
│   CollectionPermissionMiddleware(svc, ":read"),              │
│   h.GetBusinessDataByModule)                                 │
└──────────────────────────────────────────────────────────────┘
```

## 9. 刮削系统架构

### 9.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         刮削系统                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   API 层                   核心引擎                              │
│  ┌──────────┐    SubmitTask    ┌──────────────────────┐        │
│  │ SubmitTask│───────────────▶ │    taskQueue         │        │
│  │ Handler   │                │  chan *ScrapeTask    │        │
│  └──────────┘                 │  (buffer=1000)       │        │
│                               └──────┬───────────────┘        │
│                                      │                          │
│              ┌───────────────────────┼──────────────────────┐  │
│              ▼                       ▼                       ▼  │
│        ┌──────────┐           ┌──────────┐           ┌──────────┐
│        │ Worker 0 │           │ Worker 1 │    ...    │ Worker N │
│        │ goroutine│           │ goroutine│           │ goroutine│
│        └────┬─────┘           └────┬─────┘           └────┬─────┘
│             │                      │                      │
│             └──────────────────────┼──────────────────────┘
│                                    ▼
│                          ┌──────────────────┐
│                          │  executeScraper   │
│                          │  exec.Command(    │
│                          │    "python",      │
│                          │    scraperPath,   │
│                          │    dataPath)      │
│                          └────────┬─────────┘
│                                   ▼
│                          ┌──────────────────┐
│                          │  saveScrapedData  │
│                          │  存入 {module}_   │
│                          │  data 集合        │
│                          └──────────────────┘
└─────────────────────────────────────────────────────────────────┘
```

### 9.2 任务状态流转

```
                        SubmitTask()
  ┌─────────┐          ┌──────────┐   worker 取出   ┌───────────┐
  │  (None)  │─────────▶│ pending  │──────────────▶│ scraping  │
  └─────────┘          └──────────┘                └─────┬─────┘
                            ▲                            │
                            │        ┌───────────────────┤
                            │        ▼ exec 成功          ▼ exec 失败
                            │  ┌─────────┐          ┌─────────┐
                            └──│ retry   │          │ failed  │
                               │ (重新入队)│          └─────────┘
                               └─────────┘               │
                                                    retry 操作
                                                    ───────────▶ pending
                                                    ┌─────────┐
                                                    │ success │
  ┌─────────┐          ┌──────────┐   worker 取出   ┌───────────┐
  │  (None)  │─────────▶│ pending  │──────────────▶│ scraping  │
  └─────────┘          └──────────┘                └─────┬─────┘
                                                     │
                                        ┌────────────┴────────────┐
                                        ▼ exec 成功                ▼ exec 失败
                                   ┌─────────┐              ┌─────────┐
                                   │ success │              │ failed  │
                                   │ 存储结果 │              └────┬────┘
                                   └─────────┘                   │
                                                            retry 操作
                                                            ───────────▶ pending
```

### 9.3 Worker 处理流程

```
processTask(task, workerID):
  1. 更新状态 → scraping，记录 started_at
  2. executeScraper(scraperPath, dataPath)
     2a. 检查 scraper 文件存在
     2b. 检查 data 文件存在
     2c. exec.Command("python", scraperPath, dataPath)
     2d. JSON 解析输出: { success, data, error }
  3. 成功 → saveScrapedData(task, result)
     3a. 合并刮削属性 (scrape_path, data_path, task_id, scraped_at)
     3b. 字段验证 (非阻塞，告警但继续保存)
     3c. 存储到 {module}_data 集合
     3d. 更新 task.business_data_id
  4. 更新最终状态 → success/ failed + completed_at
```

### 9.4 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `SCRAPER_WORKERS` | 4 | 工作协程数量 |
| 队列缓冲区 | 1000 | `make(chan *ScrapeTask, 1000)` |

## 10. JQL 查询语言

### 10.1 概述

JQL (JSON Query Language) 是自定义的查询表达式语言，通过手写递归下降解析器将查询字符串转换为 MongoDB `bson.M` 过滤器。解析器位于 `pkg/jql/parser.go`。

### 10.2 词法单元

| Token 类型 | 示例 | 说明 |
|-----------|------|------|
| Field | `status`, `price` | 字段名 |
| Operator | `=`, `!=`, `>`, `<`, `>=`, `<=`, `~` | 比较运算符 |
| Value | `"active"`, `100`, `3.14`, `true` | 字面量值 |
| In/NotIn | `IN`, `NOT IN` | 列表匹配 |
| Like | `~` | 正则匹配 |
| IsNull/IsNotNull | `IS NULL`, `IS NOT NULL` | 空值检查 |
| And/Or/Not | `AND`, `OR`, `NOT` | 逻辑运算符 |
| Parens | `(`, `)` | 分组括号 |
| Function | `Now()`, `StartOfWeek()` | 时间函数 |

### 10.3 语法层次

```
Expression     → OrExpression
OrExpression   → AndExpression ("OR" AndExpression)*
AndExpression  → NotExpression ("AND" NotExpression)*
NotExpression  → "NOT"? PrimaryExpression
PrimaryExpression → "(" Expression ")" | Condition
Condition      → Field Operator Value
               | Field "IN" "(" ValueList ")"
               | Field "NOT IN" "(" ValueList ")"
               | Field "IS NULL"
               | Field "IS NOT NULL"
```

### 10.4 运算符 → MongoDB 映射

| JQL 运算符 | MongoDB 操作 | 说明 |
|-----------|-------------|------|
| `=` | `{field: value}` | 等于（value=nil → $exists:false） |
| `!=` | `{field: {$ne: value}}` | 不等于 |
| `>` | `{field: {$gt: value}}` | 大于 |
| `<` | `{field: {$lt: value}}` | 小于 |
| `>=` | `{field: {$gte: value}}` | 大于等于 |
| `<=` | `{field: {$lte: value}}` | 小于等于 |
| `~` | `{field: {$regex: value, $options: "i"}}` | 模糊匹配 |
| `IN (...)` | `{field: {$in: values}}` | 在列表中 |
| `NOT IN (...)` | `{field: {$nin: values}}` | 不在列表中 |
| `IS NULL` | `{field: {$exists: false}}` | 值为空 |
| `IS NOT NULL` | `{field: {$exists: true}}` | 值不为空 |

### 10.5 内置函数

| 函数 | 返回值 |
|------|--------|
| `Now()` | 当前时间 |
| `StartOfDay()` | 当天 00:00:00 |
| `EndOfDay()` | 当天 23:59:59.999... |
| `StartOfWeek()` | 本周一 00:00:00 |
| `EndOfWeek()` | 本周日 23:59:59.999... |
| `StartOfMonth()` | 本月 1 日 00:00:00 |
| `EndOfMonth()` | 本月最后一天 23:59:59.999... |
| `CurrentUser()` | 返回 "currentUser()" 字符串（后续处理） |

### 10.6 查询示例

```jql
status = "active" AND price > 100
name ~ "重要" OR title ~ "产品"
(status = "active") AND (price > 100 OR price < 50)
created > StartOfWeek() AND module = "movie"
status IN ("active", "pending") AND assignee IS NOT NULL
category NOT IN ("deleted", "archived")
```

### 10.7 字段名前缀转换

在业务数据查询中，JQL 解析结果会经过 `prefixCustomFields()` 函数处理：非系统字段名（不在预定义系统字段列表中）会自动添加 `custom_fields.` 前缀，因为业务数据的自定义字段存储在 `custom_fields` 嵌套文档中。

**系统字段名**:
```
_id, module, description, created_at, updated_at,
created_by, updated_by, data_path, file_path, custom_fields
```

## 11. 日志系统

### 11.1 架构

使用 `zerolog` 作为结构化日志引擎，`lumberjack` 提供日志文件自动轮转。

```
┌────────────────────────────────────────────────────────┐
│                    日志系统                              │
├────────────────────────┬───────────────────────────────┤
│   Console (stdout)     │    File (lumberjack)           │
│   zerolog.ConsoleWriter│    logs/http.log               │
│                        │    logs/app.log                │
└────────────────────────┴───────────────────────────────┘
```

### 11.2 日志配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `LOG_LEVEL` | info | 日志级别 (debug/info/warn/error) |
| `LOG_HTTP_FILE` | logs/http.log | HTTP 日志文件路径 |
| `LOG_MAX_SIZE` | 100 | 单个日志文件最大 MB |
| `LOG_MAX_BACKUPS` | 5 | 保留日志文件数 |
| `LOG_MAX_AGE` | 30 | 日志保留天数 |

### 11.3 HTTP 日志中间件

`LoggerMiddleware()` 记录每个请求的：
- 请求方法、路径、状态码
- 请求处理时长
- 客户端 IP

## 12. 安全设计

### 12.1 认证
- JWT (HMAC-SHA256) 无状态认证
- Token 含 user_id/roles/permissions
- 默认有效期 24h，刷新窗口 720h

### 12.2 密码安全
- bcrypt 哈希存储 (cost=10)
- 接口响应中始终清除 `password` 字段
- 密码最小长度 8 位

### 12.3 授权
- 双层 RBAC：系统级（全局资源）+ 集合级（模块粒度）
- 超级管理员 (`system:admin`) 绕过所有权限检查
- 通配符权限支持 (`user:*` 匹配所有 user 操作)

### 12.4 数据库安全
- 业务数据库 (datacenter) 与权限数据库 (rbac) 物理隔离
- 软删除机制，数据可恢复
- 审计日志记录集合操作

### 12.5 中间件安全
- CORS 中间件允许跨域
- Gin Recovery 捕获 panic
- 所有受保护路由强制 JWT 验证

## 13. 环境变量配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_HOST` | 0.0.0.0 | 服务监听地址 |
| `SERVER_PORT` | 8080 | 服务监听端口 |
| `JWT_SECRET` | your-secret-key | JWT 签名密钥 |
| `JWT_EXPIRATION` | 24 | Token 有效期（小时） |
| `JWT_REFRESH_EXPIRATION` | 720 | 刷新 Token 有效期（小时） |
| `MONGODB_URI` | mongodb://localhost:27017 | 业务数据库连接 |
| `MONGODB_DATABASE` | datacenter | 业务数据库名 |
| `MONGODB_RBAC_URI` | mongodb://localhost:27017 | RBAC 数据库连接 |
| `MONGODB_RBAC_DATABASE` | rbac | RBAC 数据库名 |
| `LOG_LEVEL` | info | 日志级别 |
| `LOG_HTTP_FILE` | logs/http.log | HTTP 日志文件 |
| `LOG_MAX_SIZE` | 100 | 日志最大 MB |
| `LOG_MAX_BACKUPS` | 5 | 日志备份数 |
| `LOG_MAX_AGE` | 30 | 日志保留天数 |
| `SCRAPER_WORKERS` | 4 | 刮削工作协程数 |

## 14. 前端架构

### 14.1 技术栈

| 技术 | 用途 |
|------|------|
| React 18 | UI 框架 |
| TypeScript 5 | 类型安全 |
| Ant Design 5 | 企业级 UI 组件 |
| Vite 5 | 构建工具 |
| React Router v6 | 客户端路由 |
| Zustand | 状态管理 |
| Axios | HTTP 客户端 |

### 14.2 页面结构

```
LoginPage                     # 登录页（公开）
AdminLayout                   # 管理后台布局（受保护）
├── UserManagement            # 用户管理
├── RoleManagement            # 角色管理
├── PermissionManagement      # 权限管理
├── CustomFieldsPage          # 自定义字段管理
├── ScraperCenter             # 刮削任务中心
├── SearchPage                # 高级搜索（JQL）
├── CollectionQueryPage       # 集合数据查询
├── DeletedScraperPage        # 已删除刮削任务
└── TestResultsPage           # 测试结果页
```

### 14.3 服务层

前端 `services/` 目录封装了所有后端 API 调用：

| 服务文件 | 对应后端 API |
|----------|-------------|
| `api.ts` | Axios 实例（base URL, 拦截器, Token 注入） |
| `auth.ts` | /api/auth/* |
| `user.ts` | /api/users/* |
| `rbac.ts` | /api/roles/*, /api/permissions/* |
| `business.ts` | /api/business/*, /api/collection-data/* |
| `scraper.ts` | /api/scraper/*, /api/deleted-scraper/* |
| `jql.ts` | JQL 查询相关 |

### 14.4 状态管理

使用 Zustand 管理全局认证状态：

```typescript
// authStore 管理:
// - token: JWT Token
// - user: 当前用户信息
// - login() / logout() 操作
```

## 15. 部署架构

```
                  ┌──────────────────────┐
                  │    Nginx (可选)       │
                  │  反向代理 / 静态资源   │
                  │  SSL 终止             │
                  └──────────┬───────────┘
                             │
              ┌──────────────┴──────────────┐
              ▼                             ▼
     ┌─────────────────┐          ┌─────────────────┐
     │  Backend :8080   │          │  Frontend (dist) │
     │  Go + Gin        │          │  静态文件         │
     └────────┬────────┘          └─────────────────┘
              │
              ▼
     ┌───────────────────────┐
     │      MongoDB           │
     │  ├── datacenter (业务)  │
     │  └── rbac (权限)       │
     └───────────────────────┘
```

## 16. 设计模式总结

| 模式 | 应用位置 | 说明 |
|------|----------|------|
| **依赖注入** | `cmd/server/main.go` | Handler 构造函数接收所有依赖接口 |
| **接口隔离** | `internal/scraper/scraper.go` | `Scraper` 接口定义，`scraper` 结构体实现 |
| **策略模式** | `pkg/jql/parser.go` | 不同 Token 类型对应不同解析策略 |
| **中间件链** | `internal/api/handlers.go` | Gin 中间件管道处理认证/授权/日志 |
| **对象组合** | `internal/models/models.go` | `BaseModel` 组合到各业务模型 |
| **工作池** | `internal/scraper/scraper.go` | goroutine worker pool + channel 队列 |
| **工厂函数** | 各包 | `NewXxx()` 构造函数模式 |

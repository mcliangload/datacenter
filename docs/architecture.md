# 数据中心系统架构设计文档

## 1. 系统总体架构概述

本数据中心系统采用分层架构设计，基于Go语言、Gin框架和MongoDB构建，实现企业级数据管理、用户权限控制、日志记录和业务数据处理等核心功能。系统遵循RESTful API设计风格，提供高可用性、可扩展性和安全性。

### 1.1 架构分层

```
┌─────────────────────────────────────────────────────────────┐
│                      客户端层 (Client Layer)                  │
│                   React + TypeScript + Ant Design            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼ HTTPS
┌─────────────────────────────────────────────────────────────┐
│                      API层 (API Layer)                       │
│                  Gin Router + Handlers                       │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐    │
│  │ Auth API   │ Business API│ Scraper API│ RBAC API   │    │
│  └─────────────┴─────────────┴─────────────┴─────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   业务数据库     │ │   RBAC数据库     │ │   日志系统       │
│  MongoDB        │ │  MongoDB        │ │  Zerolog       │
│  datacenter     │ │  rbac           │ │  Lumberjack    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

### 1.2 核心功能模块

| 模块 | 描述 |
|------|------|
| 用户权限系统 | 基于RBAC模型的权限管理，用户、角色、权限的完整CRUD操作 |
| 认证授权 | 基于JWT Token的认证机制，支持Token刷新和过期策略 |
| 业务数据管理 | 按模块划分的业务数据存储，支持自定义字段、软删除和数据恢复 |
| 数据刮削系统 | 异步并发处理数据刮削任务，支持任务状态管理和重试机制 |
| 集合管理 | 动态创建和管理MongoDB集合，支持索引管理 |
| 日志系统 | 结构化日志，支持HTTP日志、应用日志分离 |

## 2. 技术栈

| 分类 | 技术/框架 | 版本 | 说明 |
|------|----------|------|------|
| **后端** | | | |
| 语言 | Go | 1.20+ | 编译型语言，高性能，适合高并发 |
| 框架 | Gin | v1.9.1 | 轻量级Web框架，性能出色 |
| 数据库 | MongoDB | 6.0+ | 文档型数据库，支持动态字段 |
| 认证 | JWT | v5.0.0 | 无状态认证 |
| 日志 | zerolog + lumberjack | - | 高性能结构化日志 |
| 密码 | bcrypt | v0.14.0 | 安全密码哈希 |
| **前端** | | | |
| 框架 | React | 18+ | UI框架 |
| 语言 | TypeScript | 5+ | 类型安全 |
| UI库 | Ant Design | 5+ | 企业级UI组件 |
| 构建 | Vite | 5+ | 快速构建工具 |
| 路由 | React Router | v6 | 路由管理 |
| 状态 | Zustand | - | 轻量状态管理 |

## 3. 模块划分及职责

### 3.1 后端模块结构

```
datacenter/
├── cmd/
│   └── server/
│       └── main.go              # 应用入口
├── internal/
│   ├── api/
│   │   └── handlers.go          # HTTP处理器 (所有RESTful接口)
│   ├── auth/
│   │   ├── jwt.go              # JWT服务
│   │   ├── middleware.go        # 认证中间件
│   │   └── test/               # JWT测试
│   ├── logger/
│   │   ├── logger.go           # 日志初始化
│   │   └── middleware.go       # HTTP日志中间件
│   ├── models/
│   │   └── models.go           # 数据模型定义
│   ├── scraper/
│   │   └── scraper.go          # 刮削任务处理系统
│   └── storage/
│       ├── mongodb.go          # MongoDB连接
│       ├── mongodb_storage.go  # 业务数据存储
│       └── rbac_storage.go     # RBAC存储
├── pkg/
│   ├── jql/
│   │   └── parser.go           # JQL查询解析器
│   └── rbac/
│       └── rbac.go             # RBAC权限服务
└── docs/                       # 文档
```

### 3.2 模块职责说明

| 模块 | 路径 | 职责 |
|------|------|------|
| **后端** | | |
| API Handlers | internal/api | 处理所有HTTP请求，实现RESTful接口 |
| Auth | internal/auth | JWT Token生成验证、密码加密、认证中间件 |
| Logger | internal/logger | zerolog日志初始化、HTTP请求日志中间件 |
| Models | internal/models | 定义User、Role、Permission、BusinessData、ScrapeTask等数据结构 |
| Storage | internal/storage | MongoDB数据访问层，业务和RBAC分别独立存储 |
| Scraper | internal/scraper | 刮削任务队列、工作协程池、任务状态管理 |
| JQL Parser | pkg/jql | JQL查询语句解析和转换 |
| RBAC | pkg/rbac | 权限检查、用户权限获取、通配符匹配 |

## 4. 数据库架构

### 4.1 双数据库设计

系统使用两个独立的MongoDB数据库实现权限隔离：

```
┌─────────────────────────────────────────────────────────────┐
│                     MongoDB 集群                             │
├─────────────────────────┬─────────────────────────────────┤
│    datacenter 数据库      │         rbac 数据库              │
├─────────────────────────┼─────────────────────────────────┤
│ collections             │ users                            │
│ field_definitions       │ permissions                      │
│ scrape_tasks            │ roles                            │
│ deleted_scrape_tasks    │                                  │
│ {module}_data (动态)    │                                  │
│ deleted_data            │                                  │
└─────────────────────────┴─────────────────────────────────┘
```

### 4.2 业务数据库集合 (datacenter)

| 集合名 | 说明 | 动态创建 |
|--------|------|----------|
| collections | 集合元数据 | 否 |
| field_definitions | 字段定义 | 否 |
| scrape_tasks | 刮削任务 | 否 |
| deleted_scrape_tasks | 已删除刮削任务(软删除) | 否 |
| {module}_data | 各模块业务数据 | 是 |
| deleted_data | 已删除数据(软删除) | 否 |

### 4.3 RBAC数据库集合 (rbac)

| 集合名 | 说明 |
|--------|------|
| users | 用户信息 (含role_ids数组) |
| permissions | 权限定义 |
| roles | 角色定义 (含permission_ids数组) |

## 5. 数据模型详细设计

### 5.1 业务数据库 (datacenter)

#### 5.1.1 collections 集合

集合元数据，存储模块信息。

```json
{
  "_id": ObjectId,
  "module": "movie",
  "description": "电影数据模块",
  "datatype_owner": "admin",
  "collection_name": "movie_data",
  "created_by": "admin",
  "created_at": ISODate("2024-01-15T10:00:00Z"),
  "updated_by": "admin",
  "updated_at": ISODate("2024-01-15T10:00:00Z")
}
```

**索引**:
- `{ module: 1 }` - 唯一索引

#### 5.1.2 field_definitions 集合

字段定义，存储各模块的自定义字段信息。

```json
{
  "_id": ObjectId,
  "module": "movie",
  "field_name": "title",
  "field_type": "string",
  "description": "电影标题",
  "required": false,
  "default_value": null,
  "constraints": {
    "type": "string",
    "min_length": 1,
    "max_length": 200,
    "pattern": "",
    "enum_values": []
  },
  "created_by": "admin",
  "created_at": ISODate("2024-01-15T10:00:00Z"),
  "updated_by": "admin",
  "updated_at": ISODate("2024-01-15T10:00:00Z")
}
```

**约束类型**:
- string: min_length, max_length, pattern, enum_values
- number: min, max
- array: list_min_length, list_max_length

**索引**:
- `{ module: 1, field_name: 1 }` - 复合唯一索引

#### 5.1.3 scrape_tasks 集合

刮削任务记录，关联业务数据。

```json
{
  "_id": ObjectId,
  "module": "movie",
  "data_path": "/data/movies/harry_potter.json",
  "scraper_path": "/scrapers/movie_scraper.py",
  "status": "success",
  "result": {
    "title": "Harry Potter",
    "director": "Chris Columbus"
  },
  "error_message": "",
  "started_at": ISODate("2024-01-15T10:30:00Z"),
  "completed_at": ISODate("2024-01-15T10:30:02Z"),
  "business_data_id": ObjectId("..."),
  "description": "电影数据刮削",
  "created_by": "admin",
  "created_at": ISODate("2024-01-15T10:30:00Z"),
  "updated_by": "admin",
  "updated_at": ISODate("2024-01-15T10:30:02Z")
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| module | string | 所属模块 |
| data_path | string | 原始数据文件路径 |
| scraper_path | string | 刮削器脚本路径 |
| status | string | 任务状态: pending/scraping/success/failed |
| result | object | 刮削结果详情 |
| error_message | string | 失败时的错误信息 |
| started_at | datetime | 开始时间 |
| completed_at | datetime | 完成时间 |
| business_data_id | ObjectId | 关联的业务数据ID |
| description | string | 任务描述 |

**索引**:
- `{ module: 1, status: 1 }` - 复合索引
- `{ created_at: -1 }` - 降序索引

#### 5.1.4 {module}_data 集合 (动态集合)

各模块的业务数据，刮削完成后数据存储位置。

```json
{
  "_id": ObjectId,
  "module": "movie",
  "description": "刮削数据 - /data/movies/harry_potter.json",
  "custom_fields": {
    "title": "Harry Potter and the Sorcerer's Stone",
    "director": "Chris Columbus",
    "year": 2001,
    "genre": ["Fantasy", "Adventure"],
    "rating": 7.6,
    "scrape_path": "/scrapers/movie_scraper.py",
    "data_path": "/data/movies/harry_potter.json",
    "task_id": "任务ID",
    "scraped_at": ISODate("2024-01-15T10:30:02Z")
  },
  "file_path": "/data/movies/harry_potter.json",
  "created_by": "admin",
  "created_at": ISODate("2024-01-15T10:30:02Z"),
  "updated_by": "admin",
  "updated_at": ISODate("2024-01-15T10:30:02Z")
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| module | string | 所属模块 |
| description | string | 数据描述 |
| custom_fields | object | 自定义字段，包含刮削属性 |
| file_path | string | 原始数据文件路径 |
| scrape_path | string | 刮削器脚本路径 (在custom_fields中) |
| task_id | string | 刮削任务ID (在custom_fields中) |
| scraped_at | datetime | 刮削完成时间 (在custom_fields中) |

#### 5.1.5 deleted_data 集合

软删除的数据记录。

```json
{
  "_id": ObjectId,
  "module": "movie",
  "original_id": ObjectId("..."),
  "description": "电影数据",
  "custom_fields": {...},
  "file_path": "/data/movies/harry_potter.json",
  "deleted_at": ISODate("2024-01-16T10:00:00Z"),
  "created_by": "admin",
  "created_at": ISODate("2024-01-15T10:30:02Z"),
  "updated_by": "admin",
  "updated_at": ISODate("2024-01-16T10:00:00Z")
}
```

**索引**:
- `{ module: 1 }` - 模块索引
- `{ original_id: 1 }` - 原始ID索引
- `{ deleted_at: 1 }` - 删除时间索引

#### 5.1.6 deleted_scrape_tasks 集合

软删除的刮削任务记录。

```json
{
  "_id": ObjectId,
  "module": "movie",
  "original_id": ObjectId("..."),
  "data_path": "/data/movies/harry_potter.json",
  "scraper_path": "/scrapers/movie_scraper.py",
  "status": "success",
  "result": {...},
  "error_message": "",
  "started_at": ISODate("2024-01-15T10:30:00Z"),
  "completed_at": ISODate("2024-01-15T10:30:02Z"),
  "business_data_id": ObjectId("..."),
  "deleted_at": ISODate("2024-01-16T10:00:00Z"),
  "created_at": ISODate("2024-01-15T10:30:00Z"),
  "updated_at": ISODate("2024-01-16T10:00:00Z")
}
```

### 5.2 RBAC数据库 (rbac)

#### 5.2.1 users 集合

用户信息。

```json
{
  "_id": ObjectId,
  "username": "admin",
  "password": "$2a$10$...",
  "email": "admin@datacenter.local",
  "role_ids": ["role_id_1", "role_id_2"],
  "created_by": "system",
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_by": "system",
  "updated_at": ISODate("2024-01-01T00:00:00Z")
}
```

**索引**:
- `{ username: 1 }` - 唯一索引
- `{ email: 1 }` - 唯一索引

#### 5.2.2 permissions 集合

权限定义。

```json
{
  "_id": ObjectId,
  "name": "用户管理",
  "code": "user:write",
  "description": "管理系统用户账户",
  "created_by": "system",
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_by": "system",
  "updated_at": ISODate("2024-01-01T00:00:00Z")
}
```

**索引**:
- `{ code: 1 }` - 唯一索引

#### 5.2.3 roles 集合

角色定义。

```json
{
  "_id": ObjectId,
  "name": "管理员",
  "code": "admin",
  "description": "系统管理员",
  "permission_ids": ["perm_id_1", "perm_id_2"],
  "created_by": "system",
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_by": "system",
  "updated_at": ISODate("2024-01-01T00:00:00Z")
}
```

**索引**:
- `{ code: 1 }` - 唯一索引

## 6. API架构

### 6.1 API分组

```
/api
├── public (无需认证)
│   ├── POST /auth/login          # 用户登录
│   └── POST /auth/register       # 用户注册
│
└── protected (需要JWT认证)
    ├── /users                   # 用户管理CRUD
    ├── /permissions             # 权限管理CRUD
    ├── /roles                  # 角色管理CRUD
    ├── /fields                 # 字段定义CRUD
    ├── /business               # 业务数据CRUD
    ├── /deleted                # 已删除数据
    ├── /scraper               # 刮削任务管理
    ├── /deleted-scraper       # 已删除刮削任务
    └── /collections            # 集合管理
```

### 6.2 API端点清单

#### 认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/login | 用户登录 |
| POST | /api/auth/register | 用户注册 |

#### 用户管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/users | 创建用户 |
| GET | /api/users | 获取用户列表 (分页) |
| GET | /api/users/:id | 获取用户详情 |
| PUT | /api/users/:id | 更新用户 |
| DELETE | /api/users/:id | 删除用户 |
| POST | /api/users/:id/roles | 分配角色 |
| DELETE | /api/users/:id/roles/:roleId | 移除角色 |
| GET | /api/users/:id/roles | 获取用户角色 |

#### 权限管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/permissions | 创建权限 |
| GET | /api/permissions | 获取权限列表 (分页) |
| GET | /api/permissions/:id | 获取权限详情 |
| PUT | /api/permissions/:id | 更新权限 |
| DELETE | /api/permissions/:id | 删除权限 |

#### 角色管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/roles | 创建角色 |
| GET | /api/roles | 获取角色列表 (分页) |
| GET | /api/roles/:id | 获取角色详情 |
| PUT | /api/roles/:id | 更新角色 |
| DELETE | /api/roles/:id | 删除角色 |
| POST | /api/roles/:id/permissions | 分配权限 |
| DELETE | /api/roles/:id/permissions/:permissionId | 移除权限 |
| GET | /api/roles/:id/permissions | 获取角色权限 |

#### 业务数据接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/business | 创建业务数据 |
| GET | /api/business/module/:module | 按模块查询 (分页,JQL查询) |
| GET | /api/business/module/:module/:id | 获取详情 |
| PUT | /api/business/module/:module/:id | 更新 |
| DELETE | /api/business/module/:module/:id | 删除(软删除) |

#### 集合管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/collections | 创建集合 |
| GET | /api/collections | 获取集合列表 |
| GET | /api/collections/:module | 获取集合详情 |
| PUT | /api/collections/:module | 更新集合 |
| DELETE | /api/collections/:module | 删除集合 |
| POST | /api/collections/:module/indexes | 创建索引 |
| GET | /api/collections/:module/indexes | 获取索引列表 |
| DELETE | /api/collections/:module/indexes/:name | 删除索引 |

## 7. 刮削系统架构

### 7.1 系统组件

```
┌─────────────────────────────────────────────────────────────┐
│                      刮削系统                                 │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│  │  任务队列    │───▶│  Worker 1   │    │  Worker N   │    │
│  │  (Channel)   │    │  goroutine  │    │  goroutine  │    │
│  └─────────────┘    └─────────────┘    └─────────────┘    │
│         │                  │                  │           │
│         └──────────────────┼──────────────────┘           │
│                            ▼                               │
│         ┌─────────────────────────────────┐              │
│         │         MongoDB                   │              │
│         │  ┌─────────────┐ ┌───────────┐  │              │
│         │  │scrape_tasks │ │movie_data │  │              │
│         │  └─────────────┘ └───────────┘  │              │
│         └─────────────────────────────────┘              │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 任务状态流

```
┌──────────┐    submit    ┌───────────┐   finish   ┌─────────┐
│  None    │─────────────▶│ pending   │──────────▶│ success │
└──────────┘              └───────────┘            └─────────┘
                              │
                              ▼
                         ┌─────────┐
                         │ scraping│
                         └─────────┘
                              │
                              │   │ (worker从队列取出)
                              │   └────── fail ───────────┘
                              ▼
                         ┌─────────┐
                         │ failed  │
                         └─────────┘
                              │
                              └───── retry ────▶ pending
```

### 7.3 刮削流程

1. **提交任务**: 用户通过 `/api/scraper/upload` 提交刮削任务
2. **验证模块**: 检查模块集合是否存在
3. **创建任务记录**: 在 `scrape_tasks` 集合中创建任务记录，状态为 `pending`
4. **加入任务队列**: 任务进入工作协程池的 Channel 队列
5. **执行刮削**: Worker 协程从 Channel 取出任务，执行刮削器脚本
6. **更新状态**: 根据执行结果更新任务状态为 `scraping` -> `success`/`failed`
7. **存储结果**: 成功时，将刮削结果存储到 `{module}_data` 集合，并更新任务的 `business_data_id`
8. **任务重试**: 用户可以重新提交同一数据路径的刮削任务进行重复刮削

### 7.4 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| SCRAPER_WORKERS | 4 | 工作协程数量 |
| SCRAPER_QUEUE_SIZE | 1000 | 任务队列大小 |

## 8. RBAC权限模型

### 8.1 数据模型

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│    User     │       │    Role     │       │  Permission │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ _id         │       │ _id         │       │ _id         │
│ username    │       │ name        │       │ name        │
│ email       │       │ code        │       │ code        │
│ password    │       │ description │       │ description │
│ role_ids[]  │◀─────▶│ permission_ │──────▶│             │
│             │  N:M  │   ids[]    │  N:M  │             │
└─────────────┘       └─────────────┘       └─────────────┘
```

### 8.2 权限代码格式

系统使用基于资源的权限代码格式：

| 权限代码 | 描述 |
|----------|------|
| user:* | 用户完全控制 |
| user:read | 查看用户 |
| user:write | 管理用户 |
| role:* | 角色完全控制 |
| role:read | 查看角色 |
| role:write | 管理角色 |
| permission:* | 权限完全控制 |
| permission:read | 查看权限 |
| permission:write | 管理权限 |
| data:* | 数据完全控制 |
| data:read | 查看数据 |
| data:write | 管理数据 |
| field:* | 字段完全控制 |
| field:read | 查看字段 |
| field:write | 管理字段 |
| scrape:* | 刮削完全控制 |
| scrape:read | 查看刮削任务 |
| scrape:write | 管理刮削任务 |
| collection:* | 集合完全控制 |
| collection:read | 查看集合 |
| collection:write | 管理集合 |

### 8.3 通配符权限匹配

RBAC服务支持通配符权限匹配：
- `user:*` 可以匹配 `user:read`、`user:write` 等所有 user 模块的权限
- `data:*` 可以匹配 `data:read`、`data:write` 等所有 data 模块的权限

## 9. 分页查询

### 9.1 分页响应格式

所有列表查询接口都返回分页响应，格式如下:

```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "pageSize": 10
}
```

### 9.2 分页参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 当前页码 |
| pageSize | int | 10 | 每页数量 |

**总页数计算**: `totalPages = Math.ceil(total / pageSize)`

## 10. 日志系统

### 10.1 日志分层

| 日志类型 | 文件 | 说明 |
|----------|------|------|
| HTTP日志 | logs/http.log | API请求响应记录 |
| 应用日志 | logs/app.log | 程序运行日志 |

### 10.2 日志配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| LOG_LEVEL | info | 日志级别 |
| LOG_HTTP_FILE | logs/http.log | HTTP日志文件路径 |
| LOG_MAX_SIZE | 100 | 单个日志文件最大MB |
| LOG_MAX_BACKUPS | 5 | 保留日志文件数 |
| LOG_MAX_AGE | 30 | 日志保留天数 |

### 10.3 日志格式

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "caller": "handler.go:123",
  "message": "Request processed",
  "method": "POST",
  "path": "/api/users",
  "status": 200,
  "duration": "45ms"
}
```

## 11. 安全设计

### 11.1 认证机制

- JWT Token认证，默认有效期24小时
- Token包含用户ID和角色信息
- 支持Token刷新机制，默认刷新有效期720小时(30天)

### 11.2 密码安全

- 使用bcrypt加密存储
- 密码最小长度8位

### 11.3 数据库隔离

- 业务数据库和RBAC数据库物理隔离
- 使用不同连接凭据
- 业务数据库用户无RBAC数据库访问权限

## 12. 环境变量配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| SERVER_HOST | 0.0.0.0 | 服务监听地址 |
| SERVER_PORT | 8080 | 服务监听端口 |
| JWT_SECRET | your-secret-key | JWT密钥 |
| JWT_EXPIRATION | 24 | Token有效期(小时) |
| JWT_REFRESH_EXPIRATION | 720 | 刷新Token有效期(小时) |
| MONGODB_URI | mongodb://localhost:27017 | 业务数据库连接 |
| MONGODB_DATABASE | datacenter | 业务数据库名 |
| MONGODB_RBAC_URI | mongodb://localhost:27017 | RBAC数据库连接 |
| MONGODB_RBAC_DATABASE | rbac | RBAC数据库名 |
| LOG_LEVEL | info | 日志级别 |
| LOG_HTTP_FILE | logs/http.log | HTTP日志文件 |
| LOG_MAX_SIZE | 100 | 日志最大MB |
| LOG_MAX_BACKUPS | 5 | 日志备份数 |
| LOG_MAX_AGE | 30 | 日志保留天数 |
| SCRAPER_WORKERS | 4 | 刮削工作协程数 |

## 13. 部署架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Nginx (可选)                          │
│                    负载均衡 / SSL终止                        │
└─────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
            ┌───────────────┐   ┌───────────────┐
            │  Backend 1    │   │  Backend N    │
            │  datacenter   │   │  datacenter   │
            │  :8080        │   │  :8080        │
            └───────────────┘   └───────────────┘
                    │                   │
                    └─────────┬─────────┘
                              ▼
                    ┌───────────────────┐
                    │     MongoDB       │
                    │ ┌───────────────┐ │
                    │ │  datacenter   │ │
                    │ ├───────────────┤ │
                    │ │     rbac      │ │
                    │ └───────────────┘ │
                    └───────────────────┘
```

# 数据中心系统架构设计文档

## 文档版本信息

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2024-01-15 | 初始版本 |
| 2.0 | 2026-04-19 | 重大更新：新增数据导出功能、完整的Mermaid图表 |

---

## 目录

1. [系统概述](#1-系统概述)
2. [系统架构](#2-系统架构)
3. [技术栈](#3-技术栈)
4. [模块划分](#4-模块划分)
5. [数据库架构](#5-数据库架构)
6. [数据模型](#6-数据模型)
7. [API架构](#7-api架构)
8. [核心业务流程](#8-核心业务流程)
9. [RBAC权限模型](#9-rbac权限模型)
10. [导出功能设计](#10-导出功能设计)

---

## 1. 系统概述

### 1.1 项目简介

数据中心系统是一个基于Go和React的企业级数据管理平台，提供用户认证、RBAC权限管理、业务数据管理和数据刮削功能。系统采用微服务化的分层架构设计，支持多模块数据管理、软删除、数据恢复等企业级特性。

### 1.2 核心特性

- **多模块数据管理**：支持图书、电影、音乐、游戏等多种数据模块
- **RBAC权限控制**：基于角色的访问控制，支持细粒度权限管理
- **数据刮削系统**：异步并发处理数据刮削任务，支持任务状态管理和重试
- **软删除与恢复**：支持数据软删除和恢复，防止误删
- **JQL查询**：自定义查询语言，支持灵活的数据检索
- **数据导出**：支持多格式数据导出（JSON、CSV、Excel等）
- **双数据库架构**：业务数据和RBAC数据分离存储，提高安全性

---

## 2. 系统架构

### 2.1 整体架构图

```mermaid
graph TB
    subgraph 客户端层["客户端层 Client Layer"]
        FE["React + TypeScript + Ant Design"]
    end

    subgraph 网关层["网关层 Gateway"]
        GW["Nginx/反向代理"]
    end

    subgraph 服务层["服务层 Service Layer"]
        subgraph API层["API Layer - Gin Framework"]
            AUTH["认证模块<br/>/api/auth/*"]
            RBAC["权限模块<br/>/api/users/* /api/roles/* /api/permissions/*"]
            BIZ["业务模块<br/>/api/business/* /api/fields/*"]
            SCRAPER["刮削模块<br/>/api/scraper/*"]
            EXPORT["导出模块<br/>/api/export/*"]
        end
    end

    subgraph 数据层["数据层 Data Layer"]
        subgraph 业务数据库["datacenter 数据库"]
            BIZ_COLLECTIONS["collections<br/>集合元数据"]
            BIZ_FIELDS["field_definitions<br/>字段定义"]
            BIZ_SCRAPE_TASKS["scrape_tasks<br/>刮削任务"]
            BIZ_DATA["{module}_data<br/>动态业务数据"]
            BIZ_DELETED["deleted_data<br/>已删除数据"]
        end

        subgraph RBAC数据库["rbac 数据库"]
            RBAC_USERS["users<br/>用户信息"]
            RBAC_PERMS["permissions<br/>权限定义"]
            RBAC_ROLES["roles<br/>角色定义"]
        end
    end

    subgraph 基础设施层["基础设施层"]
        LOG["Zerolog + Lumberjack<br/>日志系统"]
        JWT["JWT认证服务"]
        SCRAPER_ENGINE["刮削引擎"]
    end

    FE --> GW
    GW --> AUTH
    GW --> RBAC
    GW --> BIZ
    GW --> SCRAPER
    GW --> EXPORT

    AUTH --> RBAC_USERS
    RBAC --> RBAC_PERMS
    RBAC --> RBAC_ROLES
    RBAC --> RBAC_USERS

    BIZ --> BIZ_COLLECTIONS
    BIZ --> BIZ_FIELDS
    BIZ --> BIZ_DATA
    BIZ --> BIZ_DELETED

    SCRAPER --> BIZ_SCRAPE_TASKS
    SCRAPER --> BIZ_DATA
    SCRAPER --> SCRAPER_ENGINE

    EXPORT --> BIZ_DATA
    EXPORT --> LOG

    style FE fill:#e1f5ff,stroke:#01579b
    style GW fill:#fff3e0,stroke:#ef6c00
    style AUTH fill:#e8f5e9,stroke:#2e7d32
    style RBAC fill:#f3e5f5,stroke:#7b1fa2
    style BIZ fill:#fce4ec,stroke:#c2185b
    style SCRAPER fill:#fff8e1,stroke:#f9a825
    style EXPORT fill:#e0f7fa,stroke:#00838f
    style LOG fill:#f5f5f5,stroke:#9e9e9e
```

### 2.2 请求处理流程

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Gateway as 网关/Nginx
    participant Gin as Gin Router
    participant Middleware as 中间件
    participant Handler as Handler
    participant Service as Service
    participant Storage as Storage
    participant MongoDB as MongoDB

    Client->>Gateway: HTTPS Request
    Gateway->>Gin: Forward Request

    Gin->>Middleware: Apply Middlewares
    Note over Middleware: 1. CORS<br/>2. Logger<br/>3. Recovery<br/>4. Auth (JWT)

    Middleware->>Handler: Authenticated Request
    Handler->>Service: Call Business Logic

    Service->>Storage: Data Access
    Storage->>MongoDB: CRUD Operations

    MongoDB-->>Storage: Query Result
    Storage-->>Service: DTO/Entity
    Service-->>Handler: Response Data
    Handler-->>Middleware: JSON Response
    Middleware-->>Gateway: Response
    Gateway-->>Client: HTTPS Response
```

### 2.3 系统部署架构

```mermaid
graph LR
    subgraph 开发环境["开发环境 Dev"]
        FE_DEV["前端 Dev Server<br/>localhost:5173"]
        BE_DEV["后端服务<br/>localhost:9003"]
        MONGO_DEV["MongoDB<br/>localhost:27017"]
    end

    subgraph 生产环境["生产环境 Production"]
        subgraph 前端["前端集群"]
            FE_CDN["静态资源<br/>CDN分发"]
            FE_ORIGIN["Nginx<br/>负载均衡"]
        end

        subgraph 后端["后端集群"]
            BE_LB["Nginx<br/>负载均衡"]
            BE1["API Server #1"]
            BE2["API Server #2"]
            BE3["API Server #3"]
        end

        subgraph 数据库["数据库集群"]
            MONGO_PRIMARY["MongoDB Primary"]
            MONGO_REPLICA1["MongoDB Replica 1"]
            MONGO_REPLICA2["MongoDB Replica 2"]
        end
    end

    Client_dev["开发客户端"] --> FE_DEV
    FE_DEV --> BE_DEV
    BE_DEV --> MONGO_DEV

    Client_prod["生产客户端"] --> FE_CDN
    Client_prod --> FE_ORIGIN
    FE_ORIGIN --> BE_LB
    BE_LB --> BE1
    BE_LB --> BE2
    BE_LB --> BE3
    BE1 --> MONGO_PRIMARY
    BE2 --> MONGO_PRIMARY
    BE3 --> MONGO_PRIMARY
    MONGO_PRIMARY --> MONGO_REPLICA1
    MONGO_PRIMARY --> MONGO_REPLICA2

    style FE_DEV fill:#e3f2fd,stroke:#1565c0
    style BE_DEV fill:#e8f5e9,stroke:#2e7d32
    style MONGO_DEV fill:#fff3e0,stroke:#ef6c00
    style FE_CDN fill:#e3f2fd,stroke:#1565c0
    style BE_LB fill:#f3e5f5,stroke:#7b1fa2
    style MONGO_PRIMARY fill:#fff8e1,stroke:#f9a825
```

---

## 3. 技术栈

### 3.1 技术栈总览

```mermaid
graph TB
    subgraph 前端技术栈["前端 Frontend"]
        FE_FRAMEWORK["React 18+<br/>UI框架"]
        FE_LANG["TypeScript 5+<br/>类型安全"]
        FE_UI["Ant Design 5+<br/>企业级UI组件"]
        FE_BUILD["Vite 5+<br/>快速构建"]
        FE_ROUTER["React Router v6<br/>路由管理"]
        FE_STATE["Zustand<br/>状态管理"]
    end

    subgraph 后端技术栈["后端 Backend"]
        BE_LANG["Go 1.20+<br/>高性能语言"]
        BE_FRAMEWORK["Gin v1.9.1<br/>Web框架"]
        BE_DB["MongoDB 6.0+<br/>文档数据库"]
        BE_AUTH["JWT v5.0.0<br/>无状态认证"]
        BE_LOG["Zerolog + Lumberjack<br/>结构化日志"]
        BE_PASSWORD["bcrypt<br/>密码哈希"]
    end

    subgraph 基础设施["基础设施 Infrastructure"]
        INFContainer["Docker<br/>容器化"]
        INFMonitor["Prometheus + Grafana<br/>监控告警"]
        INFCICD["GitHub Actions<br/>CI/CD"]
    end

    style 前端技术栈 fill:#e3f2fd,stroke:#1565c0
    style 后端技术栈 fill:#e8f5e9,stroke:#2e7d32
    style 基础设施 fill:#fff3e0,stroke:#ef6c00
```

### 3.2 技术选型说明

| 分类 | 技术 | 版本 | 说明 |
|------|------|------|------|
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

---

## 4. 模块划分

### 4.1 项目目录结构

```mermaid
graph TD
    root["datacenter/"]
    root --> cmd["cmd/"]
    root --> configs["configs/"]
    root --> docs["docs/"]
    root --> frontend["frontend/"]
    root --> internal["internal/"]
    root --> pkg["pkg/"]
    root --> test["test/"]

    cmd --> server["server/"]
    cmd --> createdata["createdata/"]
    cmd --> generatescrapedata["generatescrapedata/"]
    cmd --> testpagination["testpagination/"]

    internal --> api["api/"]
    internal --> auth["auth/"]
    internal --> logger["logger/"]
    internal --> models["models/"]
    internal --> scraper["scraper/"]
    internal --> storage["storage/"]

    pkg --> jql["jql/"]
    pkg --> rbac["rbac/"]

    frontend --> src["src/"]
    frontend --> dist["dist/"]

    src --> pages["pages/"]
    src --> services["services/"]
    src --> stores["stores/"]
    src --> components["components/"]
    src --> types["types/"]
    src --> theme["theme/"]

    pages --> Admin["Admin/"]
    pages --> Login["Login/"]

    Admin --> AdminLayout["AdminLayout.tsx"]
    Admin --> SearchPage["SearchPage.tsx"]
    Admin --> CollectionQuery["CollectionQueryPage.tsx"]
    Admin --> CustomFields["CustomFieldsPage.tsx"]
    Admin --> UserMgmt["UserManagement.tsx"]
    Admin --> RoleMgmt["RoleManagement.tsx"]
    Admin --> PermMgmt["PermissionManagement.tsx"]
    Admin --> ScraperCenter["ScraperCenter.tsx"]
    Admin --> ModuleMgmt["ModuleManagement.tsx"]

    services --> apiSvc["api.ts"]
    services --> authSvc["auth.ts"]
    services --> bizSvc["business.ts"]
    services --> rbacSvc["rbac.ts"]
    services --> scraperSvc["scraper.ts"]
    services --> userSvc["user.ts"]

    style root fill:#f5f5f5,stroke:#9e9e9e
    style internal fill:#e8f5e9,stroke:#2e7d32
    style pkg fill:#fff3e0,stroke:#ef6c00
    style frontend fill:#e3f2fd,stroke:#1565c0
    style cmd fill:#fce4ec,stroke:#c2185b
```

### 4.2 模块职责矩阵

```mermaid
graph LR
    subgraph 后端模块["后端模块 Backend Modules"]
        API["API Handlers<br/>internal/api<br/>handlers.go"]
        AUTH["认证模块<br/>internal/auth<br/>jwt.go + middleware.go"]
        LOG["日志模块<br/>internal/logger<br/>logger.go + middleware.go"]
        MODELS["数据模型<br/>internal/models<br/>models.go"]
        STORAGE["存储层<br/>internal/storage<br/>mongodb.go + rbac_storage.go"]
        SCRAPER["刮削系统<br/>internal/scraper<br/>scraper.go"]
        JQL["JQL解析器<br/>pkg/jql<br/>parser.go"]
        RBAC_PKG["RBAC服务<br/>pkg/rbac<br/>rbac.go"]
    end

    subgraph 前端模块["前端模块 Frontend Modules"]
        PAGES["页面组件<br/>pages/Admin/*.tsx"]
        SERVICES["API服务<br/>services/*.ts"]
        STORES["状态管理<br/>stores/*.ts"]
        TYPES["类型定义<br/>types/index.ts"]
    end

    subgraph 数据库["数据库 Database"]
        BIZ_DB["datacenter<br/>业务数据库"]
        RBAC_DB["rbac<br/>权限数据库"]
    end

    API --> AUTH
    API --> MODELS
    API --> STORAGE
    API --> SCRAPER
    API --> JQL
    API --> RBAC_PKG

    AUTH --> LOG

    STORAGE --> BIZ_DB
    STORAGE --> RBAC_DB

    SCRAPER --> STORAGE
    SCRAPER --> LOG

    RBAC_PKG --> STORAGE

    PAGES --> SERVICES
    SERVICES --> API
    STORES --> SERVICES

    style API fill:#e3f2fd,stroke:#1565c0
    style AUTH fill:#e8f5e9,stroke:#2e7d32
    style STORAGE fill:#fff3e0,stroke:#ef6c00
    style SCRAPER fill:#fce4ec,stroke:#c2185b
    style RBAC_PKG fill:#f3e5f5,stroke:#7b1fa2
    style PAGES fill:#e0f7fa,stroke:#00838f
```

### 4.3 模块依赖关系

```mermaid
graph TD
    A["cmd/server<br/>应用入口"] --> B["internal/api<br/>API层"]
    A --> C["internal/auth<br/>认证层"]
    A --> D["internal/logger<br/>日志层"]
    A --> E["internal/storage<br/>存储层"]
    A --> F["internal/scraper<br/>刮削层"]

    B --> G["internal/models<br/>数据模型"]
    B --> H["pkg/jql<br/>JQL解析器"]
    B --> I["pkg/rbac<br/>RBAC服务"]

    E --> G
    E --> J["go.mongodb.org/mongo-driver<br/>MongoDB驱动"]

    C --> D
    C --> K["github.com/golang-jwt/jwt/v5<br/>JWT库"]

    D --> L["github.com/rs/zerolog<br/>日志库"]
    D --> M["gopkg.in/natefinch/lumberjack.v2<br/>日志滚动"]

    F --> D
    F --> E

    H --> J

    B --> N["github.com/gin-gonic/gin<br/>Gin框架"]

    style A fill:#f9f,stroke:#333,stroke-width:2px
    style B fill:#bbf,stroke:#333,stroke-width:2px
    style C fill:#bfb,stroke:#333,stroke-width:2px
    style D fill:#fbf,stroke:#333,stroke-width:2px
    style E fill:#ffb,stroke:#333,stroke-width:2px
    style F fill:#bff,stroke:#333,stroke-width:2px
    style G fill:#fbb,stroke:#333,stroke-width:2px
    style H fill:#bbb,stroke:#333,stroke-width:2px
```

---

## 5. 数据库架构

### 5.1 双数据库设计

```mermaid
graph TB
    subgraph MongoDB集群["MongoDB Cluster"]
        subgraph datacenter数据库["datacenter 数据库"]
            direction TB
            DC_COLLECTIONS["collections<br/>集合元数据"]
            DC_FIELDS["field_definitions<br/>字段定义"]
            DC_SCRAPE_TASKS["scrape_tasks<br/>刮削任务"]
            DC_DATA_1["book_data<br/>图书数据"]
            DC_DATA_2["movie_data<br/>电影数据"]
            DC_DATA_3["music_data<br/>音乐数据"]
            DC_DATA_N["{module}_data<br/>动态集合"]
            DC_DELETED["deleted_data<br/>已删除数据"]
            DC_DELETED_SCRAPE["deleted_scrape_tasks<br/>已删除刮削任务"]
        end

        subgraph rbac数据库["rbac 数据库"]
            direction TB
            RBAC_USERS["users<br/>用户信息"]
            RBAC_PERMISSIONS["permissions<br/>权限定义"]
            RBAC_ROLES["roles<br/>角色定义"]
        end
    end

    style datacenter数据库 fill:#e8f5e9,stroke:#2e7d32
    style rbac数据库 fill:#f3e5f5,stroke:#7b1fa2
    style DC_COLLECTIONS fill:#fff8e1
    style DC_FIELDS fill:#fff8e1
    style DC_SCRAPE_TASKS fill:#fff8e1
    style DC_DATA_1 fill:#e3f2fd
    style DC_DATA_2 fill:#e3f2fd
    style DC_DATA_3 fill:#e3f2fd
    style DC_DATA_N fill:#e3f2fd
    style DC_DELETED fill:#ffebee
    style DC_DELETED_SCRAPE fill:#ffebee
    style RBAC_USERS fill:#fce4ec
    style RBAC_PERMISSIONS fill:#fce4ec
    style RBAC_ROLES fill:#fce4ec
```

### 5.2 集合关联图

```mermaid
erDiagram
    COLLECTION ||--o| FIELD_DEFINITION : "defines"
    COLLECTION ||--o{ BUSINESS_DATA : "stores"
    MODULE ||--o| COLLECTION : "represents"
    MODULE ||--o{ FIELD_DEFINITION : "has"
    MODULE ||--o{ BUSINESS_DATA : "contains"
    USER ||--o{ BUSINESS_DATA : "creates/updates"
    USER ||--o{ FIELD_DEFINITION : "defines"
    BUSINESS_DATA ||--|| DELETED_DATA : "soft-deleted"
    SCRAPE_TASK ||--o| BUSINESS_DATA : "produces"
    SCRAPE_TASK ||--|| DELETED_SCRAPE_TASK : "soft-deleted"
    ROLE ||--o{ PERMISSION : "contains"
    USER ||--o{ ROLE : "assigned_to"

    COLLECTION {
        ObjectId _id PK
        string module UK
        string description
        string datatype_owner
        string collection_name
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    FIELD_DEFINITION {
        ObjectId _id PK
        string module FK
        string field_name
        string field_type
        string description
        object constraints
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    BUSINESS_DATA {
        ObjectId _id PK
        string module FK
        string description
        object custom_fields
        string file_path
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    DELETED_DATA {
        ObjectId _id PK
        ObjectId original_id FK
        string module FK
        string description
        object custom_fields
        string file_path
        datetime deleted_at
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    SCRAPE_TASK {
        ObjectId _id PK
        string module FK
        string data_path
        string scraper_path
        string status
        object result
        string error_message
        datetime started_at
        datetime completed_at
        ObjectId business_data_id FK
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    DELETED_SCRAPE_TASK {
        ObjectId _id PK
        ObjectId original_id FK
        string module FK
        string data_path
        string scraper_path
        string status
        object result
        string error_message
        datetime started_at
        datetime completed_at
        ObjectId business_data_id FK
        datetime deleted_at
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    USER {
        ObjectId _id PK
        string username UK
        string password
        string email
        array role_ids
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    ROLE {
        ObjectId _id PK
        string name
        string code UK
        string description
        array permission_ids
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    PERMISSION {
        ObjectId _id PK
        string name
        string code UK
        string description
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }
```

---

## 6. 数据模型

### 6.1 实体关系详细图

```mermaid
erDiagram
    USER {
        ObjectId _id PK
        string username UK
        string password
        string email
        array role_ids
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    ROLE {
        ObjectId _id PK
        string name
        string code UK
        string description
        array permission_ids
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    PERMISSION {
        ObjectId _id PK
        string name
        string code UK
        string description
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    COLLECTION {
        ObjectId _id PK
        string module UK
        string description
        string datatype_owner
        string collection_name
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    FIELD_DEFINITION {
        ObjectId _id PK
        ObjectId module FK
        string field_name
        string field_type
        string description
        object constraints
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    BUSINESS_DATA {
        ObjectId _id PK
        string module FK
        string description
        object custom_fields
        string file_path
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    DELETED_DATA {
        ObjectId _id PK
        ObjectId original_id
        string module FK
        string description
        object custom_fields
        string file_path
        datetime deleted_at
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    SCRAPE_TASK {
        ObjectId _id PK
        string module FK
        string data_path
        string scraper_path
        string status
        object result
        string error_message
        datetime started_at
        datetime completed_at
        ObjectId business_data_id
        string created_by
        datetime created_at
        string updated_by
        datetime updated_at
    }

    USER ||--o{ ROLE : "has"
    ROLE ||--o{ PERMISSION : "has"
```

### 6.2 数据模型JSON Schema

#### 6.2.1 用户模型 (User)

```json
{
  "_id": "ObjectId",
  "username": "admin",
  "password": "$2a$10$X/...",
  "email": "admin@datacenter.local",
  "role_ids": ["role_id_1", "role_id_2"],
  "created_by": "system",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_by": "system",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| _id | ObjectId | 用户唯一标识 |
| username | string | 用户名，唯一索引 |
| password | string | bcrypt加密密码 |
| email | string | 邮箱地址 |
| role_ids | array | 所属角色ID列表 |
| created_by | string | 创建者 |
| created_at | datetime | 创建时间 |
| updated_by | string | 更新者 |
| updated_at | datetime | 更新时间 |

#### 6.2.2 角色模型 (Role)

```json
{
  "_id": "ObjectId",
  "name": "管理员",
  "code": "admin",
  "description": "系统管理员，拥有所有权限",
  "permission_ids": ["perm_id_1", "perm_id_2", "..."],
  "created_by": "system",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_by": "system",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| _id | ObjectId | 角色唯一标识 |
| name | string | 角色名称 |
| code | string | 角色代码，唯一索引 |
| description | string | 角色描述 |
| permission_ids | array | 权限ID列表 |
| created_by | string | 创建者 |
| created_at | datetime | 创建时间 |
| updated_by | string | 更新者 |
| updated_at | datetime | 更新时间 |

#### 6.2.3 权限模型 (Permission)

```json
{
  "_id": "ObjectId",
  "name": "用户管理",
  "code": "user:manage",
  "description": "管理系统用户账户",
  "created_by": "system",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_by": "system",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| _id | ObjectId | 权限唯一标识 |
| name | string | 权限名称 |
| code | string | 权限代码，唯一索引 |
| description | string | 权限描述 |
| created_by | string | 创建者 |
| created_at | datetime | 创建时间 |
| updated_by | string | 更新者 |
| updated_at | datetime | 更新时间 |

#### 6.2.4 业务数据模型 (BusinessData)

```json
{
  "_id": "ObjectId",
  "module": "movie",
  "description": "Harry Potter 电影数据",
  "custom_fields": {
    "title": "Harry Potter and the Sorcerer's Stone",
    "director": "Chris Columbus",
    "year": 2001,
    "genre": ["Fantasy", "Adventure"],
    "rating": 7.6,
    "scrape_path": "/scrapers/movie_scraper.py",
    "data_path": "/data/movies/harry_potter.json",
    "task_id": "ObjectId",
    "scraped_at": "2024-01-15T10:30:02Z"
  },
  "file_path": "/data/movies/harry_potter.json",
  "created_by": "admin",
  "created_at": "2024-01-15T10:30:02Z",
  "updated_by": "admin",
  "updated_at": "2024-01-15T10:30:02Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| _id | ObjectId | 数据唯一标识 |
| module | string | 所属模块 |
| description | string | 数据描述 |
| custom_fields | object | 自定义字段集合 |
| file_path | string | 原始数据文件路径 |
| created_by | string | 创建者 |
| created_at | datetime | 创建时间 |
| updated_by | string | 更新者 |
| updated_at | datetime | 更新时间 |

#### 6.2.5 字段定义模型 (FieldDefinition)

```json
{
  "_id": "ObjectId",
  "module": "movie",
  "field_name": "title",
  "field_type": "string",
  "description": "电影标题",
  "constraints": {
    "min_length": 1,
    "max_length": 200
  },
  "created_by": "admin",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_by": "admin",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| _id | ObjectId | 字段定义唯一标识 |
| module | string | 所属模块 |
| field_name | string | 字段名称 |
| field_type | enum | 字段类型: int/float/string/list |
| description | string | 字段描述 |
| constraints | object | 字段约束 |
| created_by | string | 创建者 |
| created_at | datetime | 创建时间 |
| updated_by | string | 更新者 |
| updated_at | datetime | 更新时间 |

#### 6.2.6 刮削任务模型 (ScrapeTask)

```json
{
  "_id": "ObjectId",
  "module": "movie",
  "data_path": "/data/movies/harry_potter.json",
  "scraper_path": "/scrapers/movie_scraper.py",
  "status": "success",
  "result": {
    "items_scraped": 8,
    "duration_ms": 1523
  },
  "error_message": "",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:30:02Z",
  "business_data_id": "ObjectId",
  "created_by": "admin",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_by": "admin",
  "updated_at": "2024-01-15T10:30:02Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| _id | ObjectId | 任务唯一标识 |
| module | string | 所属模块 |
| data_path | string | 原始数据文件路径 |
| scraper_path | string | 刮削器脚本路径 |
| status | enum | 任务状态: pending/scraping/success/failed |
| result | object | 刮削结果详情 |
| error_message | string | 失败时的错误信息 |
| started_at | datetime | 开始时间 |
| completed_at | datetime | 完成时间 |
| business_data_id | ObjectId | 关联的业务数据ID |
| created_by | string | 创建者 |
| created_at | datetime | 创建时间 |
| updated_by | string | 更新者 |
| updated_at | datetime | 更新时间 |

---

## 7. API架构

### 7.1 API路由总览

```mermaid
graph TD
    API["/api"] --> AUTH["/auth"]
    API --> USERS["/users"]
    API --> PERMS["/permissions"]
    API --> ROLES["/roles"]
    API --> FIELDS["/fields"]
    API --> BIZ["/business"]
    API --> DELETED["/deleted"]
    API --> SCRAPER["/scraper"]
    API --> COLLECTIONS["/collections"]
    API --> EXPORT["/export"]

    AUTH --> AUTH_LOGIN["POST /login"]
    AUTH --> AUTH_REGISTER["POST /register"]

    USERS --> USERS_CRUD["CRUD /:id"]
    USERS --> USERS_ROLES["POST/DELETE /:id/roles"]
    USERS --> USERS_GET_ROLES["GET /:id/roles"]

    PERMS --> PERMS_CRUD["CRUD /:id"]

    ROLES --> ROLES_CRUD["CRUD /:id"]
    ROLES --> ROLES_PERMS["POST/DELETE /:id/permissions"]
    ROLES --> ROLES_GET_PERMS["GET /:id/permissions"]

    FIELDS --> FIELDS_BY_MODULE["GET /module/:module"]
    FIELDS --> FIELDS_CRUD["CRUD /:id"]

    BIZ --> BIZ_CREATE["POST /"]
    BIZ --> BIZ_BY_MODULE["GET /module/:module"]
    BIZ --> BIZ_BY_ID["CRUD /module/:module/:id"]

    DELETED --> DELETED_BY_MODULE["GET /module/:module"]
    DELETED --> DELETED_BY_ID["GET /:id"]
    DELETED --> DELETED_RECOVER["POST /:id/recover"]

    SCRAPER --> SCRAPER_UPLOAD["POST /upload"]
    SCRAPER --> SCRAPER_TASKS["CRUD /tasks/:id"]
    SCRAPER --> SCRAPER_RETRY["POST /tasks/:id/retry"]

    COLLECTIONS --> COLLECTIONS_CRUD["CRUD /:module"]
    COLLECTIONS --> COLLECTIONS_INDEXES["POST/GET/DELETE /:module/indexes"]

    EXPORT --> EXPORT_DATA["POST /data"]
    EXPORT --> EXPORT_STATUS["GET /status/:id"]

    style API fill:#e3f2fd,stroke:#1565c0
    style AUTH fill:#e8f5e9,stroke:#2e7d32
    style USERS fill:#f3e5f5,stroke:#7b1fa2
    style BIZ fill:#fce4ec,stroke:#c2185b
    style SCRAPER fill:#fff8e1,stroke:#f9a825
    style EXPORT fill:#e0f7fa,stroke:#00838f
```

### 7.2 API分组详情

| 分组 | 路径前缀 | 说明 | 认证 |
|------|----------|------|------|
| 认证 | /api/auth | 用户登录注册 | 否 |
| 用户 | /api/users | 用户CRUD和角色分配 | 是 |
| 权限 | /api/permissions | 权限CRUD | 是 |
| 角色 | /api/roles | 角色CRUD和权限分配 | 是 |
| 字段 | /api/fields | 自定义字段CRUD | 是 |
| 业务 | /api/business | 业务数据CRUD | 是 |
| 删除 | /api/deleted | 已删除数据查询和恢复 | 是 |
| 刮削 | /api/scraper | 刮削任务管理 | 是 |
| 集合 | /api/collections | 集合元数据管理 | 是 |
| 导出 | /api/export | 数据导出功能 | 是 |

### 7.3 API端点详细清单

#### 7.3.1 认证接口

```mermaid
graph LR
    subgraph 认证接口["认证接口 /api/auth"]
        A1["POST /login<br/>用户登录"]
        A2["POST /register<br/>用户注册"]
    end

    style A1 fill:#e8f5e9,stroke:#2e7d32
    style A2 fill:#e8f5e9,stroke:#2e7d32
```

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | /api/auth/login | 用户登录 | 否 |
| POST | /api/auth/register | 用户注册 | 否 |

#### 7.3.2 用户管理接口

```mermaid
graph LR
    subgraph 用户管理["用户管理 /api/users"]
        U1["POST /<br/>创建用户"]
        U2["GET /<br/>获取用户列表"]
        U3["GET /:id<br/>获取用户详情"]
        U4["PUT /:id<br/>更新用户"]
        U5["DELETE /:id<br/>删除用户"]
        U6["POST /:id/roles<br/>分配角色"]
        U7["DELETE /:id/roles/:roleId<br/>移除角色"]
        U8["GET /:id/roles<br/>获取用户角色"]
    end

    style U1 fill:#f3e5f5,stroke:#7b1fa2
    style U2 fill:#f3e5f5,stroke:#7b1fa2
    style U3 fill:#f3e5f5,stroke:#7b1fa2
    style U4 fill:#f3e5f5,stroke:#7b1fa2
    style U5 fill:#f3e5f5,stroke:#7b1fa2
    style U6 fill:#f3e5f5,stroke:#7b1fa2
    style U7 fill:#f3e5f5,stroke:#7b1fa2
    style U8 fill:#f3e5f5,stroke:#7b1fa2
```

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

#### 7.3.3 业务数据接口

```mermaid
graph LR
    subgraph 业务数据["业务数据 /api/business"]
        B1["POST /<br/>创建业务数据"]
        B2["GET /module/:module<br/>按模块查询"]
        B3["GET /module/:module/:id<br/>获取详情"]
        B4["PUT /module/:module/:id<br/>更新数据"]
        B5["DELETE /module/:module/:id<br/>删除数据"]
    end

    style B1 fill:#fce4ec,stroke:#c2185b
    style B2 fill:#fce4ec,stroke:#c2185b
    style B3 fill:#fce4ec,stroke:#c2185b
    style B4 fill:#fce4ec,stroke:#c2185b
    style B5 fill:#fce4ec,stroke:#c2185b
```

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/business | 创建业务数据(触发刮削) |
| GET | /api/business/module/:module | 按模块查询 (分页) |
| GET | /api/business/module/:module/:id | 获取详情 |
| PUT | /api/business/module/:module/:id | 更新 |
| DELETE | /api/business/module/:module/:id | 删除(软删除) |

#### 7.3.4 数据导出接口

```mermaid
graph LR
    subgraph 导出接口["导出接口 /api/export"]
        E1["POST /data<br/>发起导出任务"]
        E2["GET /status/:id<br/>查询导出状态"]
        E3["GET /download/:id<br/>下载导出文件"]
    end

    style E1 fill:#e0f7fa,stroke:#00838f
    style E2 fill:#e0f7fa,stroke:#00838f
    style E3 fill:#e0f7fa,stroke:#00838f
```

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/export/data | 发起数据导出任务 |
| GET | /api/export/status/:id | 查询导出状态 |
| GET | /api/export/download/:id | 下载导出文件 |

---

## 8. 核心业务流程

### 8.1 用户认证流程

```mermaid
sequenceDiagram
    participant U as 用户
    participant C as 客户端
    participant API as API Server
    participant Auth as 认证模块
    participant RBAC_Store as RBAC存储
    participant JWT as JWT服务

    U->>C: 输入用户名密码
    C->>API: POST /api/auth/login
    API->>RBAC_Store: GetUserByUsername(username)
    RBAC_Store->>API: 返回用户信息
    API->>Auth: CheckPassword(password, hashedPassword)
    Auth-->>API: 验证结果
    alt 密码验证成功
        API->>JWT: GenerateToken(userID, roles, permissions)
        JWT-->>API: 返回JWT Token
        API-->>C: 200 OK { token, user }
        C-->>U: 登录成功，跳转首页
    else 密码验证失败
        API-->>C: 401 Unauthorized { error: "Invalid credentials" }
        C-->>U: 显示错误信息
    end
```

### 8.2 数据刮削流程

```mermaid
sequenceDiagram
    participant U as 用户
    participant C as 客户端
    participant API as API Server
    participant Scraper as 刮削系统
    participant Storage as 存储层
    participant DB as MongoDB

    U->>C: 上传数据文件
    C->>API: POST /api/scraper/upload
    API->>Storage: CreateScrapeTask(task)
    Storage->>DB: 插入任务记录
    DB-->>Storage: 返回
    Storage-->>API: 返回任务ID
    API-->>C: 202 Accepted { task_id }
    C-->>U: 显示提交成功

    loop 异步刮削处理
        Scraper->>Storage: GetPendingTasks()
        Storage->>DB: 查询待处理任务
        DB-->>Storage: 返回任务列表
        Storage-->>Scraper: 任务列表

        Scraper->>Scraper: 执行刮削脚本
        Scraper->>Storage: UpdateTaskStatus(running)
        Storage->>DB: 更新任务状态
        DB-->>Storage: 返回

        alt 刮削成功
            Scraper->>Storage: CreateBusinessData(data)
            Storage->>DB: 插入业务数据
            DB-->>Storage: 返回
            Scraper->>Storage: UpdateTaskStatus(success)
            Storage->>DB: 更新任务状态
        else 刮削失败
            Scraper->>Storage: UpdateTaskStatus(failed, error)
            Storage->>DB: 更新任务状态
        end
    end

    U->>C: 查询任务状态
    C->>API: GET /api/scraper/tasks/:id
    API->>Storage: GetScrapeTask(id)
    Storage->>DB: 查询任务
    DB-->>Storage: 返回任务
    Storage-->>API: 返回任务
    API-->>C: 200 OK { status, result }
    C-->>U: 显示任务结果
```

### 8.3 数据导出流程

```mermaid
flowchart TD
    A[用户请求导出数据] --> B[选择导出格式和模块]
    B --> C[指定筛选条件 JQL]
    C --> D[点击导出按钮]

    D --> E[前端调用导出API]
    E --> F{导出方式}
    F -->|同步导出| G[直接返回数据]
    F -->|异步导出| H[创建导出任务]

    H --> I[返回任务ID]
    I --> J[前端轮询任务状态]
    J --> K{任务状态}
    K -->|处理中| J
    K -->|完成| L[获取下载链接]
    K -->|失败| M[显示错误信息]

    L --> N[点击下载按钮]
    N --> O[浏览器下载文件]

    G --> P[前端处理响应]
    P --> Q[触发浏览器下载]

    style A fill:#e3f2fd,stroke:#1565c0
    style H fill:#fff8e1,stroke:#f9a825
    style L fill:#e8f5e9,stroke:#2e7d32
    style M fill:#ffebee,stroke:#c2185b
```

### 8.4 数据软删除与恢复流程

```mermaid
flowchart TD
    subgraph 删除流程["数据删除流程"]
        A1[删除请求] --> A2[验证用户权限]
        A2 --> A3{权限检查}
        A3 -->|无权限| A4[返回403 Forbidden]
        A3 -->|有权限| A5[获取原始数据]
        A5 --> A6[创建删除记录]
        A6 --> A7[插入deleted_data集合]
        A7 --> A8[删除原始数据]
        A8 --> A9[返回成功响应]
    end

    subgraph 恢复流程["数据恢复流程"]
        B1[恢复请求] --> B2[验证用户权限]
        B2 --> B3{权限检查}
        B3 -->|无权限| B4[返回403 Forbidden]
        B3 -->|有权限| B5[获取删除记录]
        B5 --> B6[创建新业务数据]
        B6 --> B7[插入业务集合]
        B7 --> B8[删除删除记录]
        B8 --> B9[返回成功响应]
    end

    subgraph 清理流程["定时清理流程"]
        C1[定时任务触发] --> C2[查询48小时前的删除记录]
        C2 --> C3[批量删除过期记录]
        C3 --> C4[清理完成]
    end

    style A1 fill:#ffebee,stroke:#c2185b
    style B1 fill:#e8f5e9,stroke:#2e7d32
    style C1 fill:#fff3e0,stroke:#ef6c00
```

### 8.5 JQL查询处理流程

```mermaid
flowchart TD
    A[接收JQL查询字符串] --> B[词法分析器]
    B --> C[Token序列]
    C --> D[语法分析器]
    D --> E{语法正确?}
    E -->|否| F[返回语法错误]
    E -->|是| G[构建抽象语法树AST]
    G --> H[遍历AST节点]
    H --> I[处理内置函数]
    I --> J[转换为MongoDB查询]
    J --> K[执行MongoDB查询]
    K --> L[返回查询结果]

    style A fill:#e3f2fd,stroke:#1565c0
    style K fill:#e8f5e9,stroke:#2e7d32
    style F fill:#ffebee,stroke:#c2185b
```

---

## 9. RBAC权限模型

### 9.1 RBAC模型结构

```mermaid
graph TB
    subgraph 核心实体["RBAC核心实体"]
        USER["用户<br/>User"]
        ROLE["角色<br/>Role"]
        PERMISSION["权限<br/>Permission"]
    end

    subgraph 关系["关系"]
        UR["用户-角色关系<br/>User-Role"]
        RP["角色-权限关系<br/>Role-Permission"]
    end

    USER --> UR
    ROLE --> UR
    ROLE --> RP
    PERMISSION --> RP

    USER -->|1:N| USER1["用户1<br/>用户2<br/>用户3..."]
    ROLE -->|1:N| ROLE1["角色1<br/>角色2<br/>角色3..."]
    PERMISSION -->|1:N| PERM1["权限1<br/>权限2<br/>权限3..."]

    style USER fill:#e3f2fd,stroke:#1565c0
    style ROLE fill:#f3e5f5,stroke:#7b1fa2
    style PERMISSION fill:#e8f5e9,stroke:#2e7d32
    style UR fill:#fff8e1,stroke:#f9a825
    style RP fill:#fff8e1,stroke:#f9a825
```

### 9.2 内置权限清单

```mermaid
graph TB
    subgraph 系统权限["系统权限 System Permissions"]
        P1["user:manage<br/>用户管理"]
        P2["role:manage<br/>角色管理"]
        P3["permission:manage<br/>权限管理"]
    end

    subgraph 数据权限["数据权限 Data Permissions"]
        P4["data:query<br/>数据查询"]
        P5["data:create<br/>数据创建"]
        P6["data:update<br/>数据更新"]
        P7["data:delete<br/>数据删除"]
    end

    subgraph 业务权限["业务权限 Business Permissions"]
        P8["datatype:define<br/>类型定义"]
        P9["collection:manage<br/>集合管理"]
        P10["scrape:manage<br/>刮削管理"]
        P11["audit:view<br/>审计查看"]
    end

    style P1 fill:#ffebee,stroke:#c2185b
    style P2 fill:#ffebee,stroke:#c2185b
    style P3 fill:#ffebee,stroke:#c2185b
    style P4 fill:#e8f5e9,stroke:#2e7d32
    style P5 fill:#e8f5e9,stroke:#2e7d32
    style P6 fill:#e8f5e9,stroke:#2e7d32
    style P7 fill:#e8f5e9,stroke:#2e7d32
    style P8 fill:#e3f2fd,stroke:#1565c0
    style P9 fill:#e3f2fd,stroke:#1565c0
    style P10 fill:#e3f2fd,stroke:#1565c0
    style P11 fill:#e3f2fd,stroke:#1565c0
```

### 9.3 内置角色定义

```mermaid
graph TB
    subgraph 角色["内置角色"]
        R1["root<br/>超级管理员<br/>全部11个权限"]
        R2["datatypeowner<br/>数据类型所有者<br/>data:* + datatype:*<br/>+ collection:* + scrape:*"]
        R3["dataowner<br/>数据所有者<br/>data:*"]
        R4["admin<br/>集合管理员<br/>data:*"]
        R5["user<br/>集合用户<br/>data:query + data:update"]
        R6["viewer<br/>只读用户<br/>data:query"]
    end

    style R1 fill:#fce4ec,stroke:#c2185b
    style R2 fill:#f3e5f5,stroke:#7b1fa2
    style R3 fill:#e8f5e9,stroke:#2e7d32
    style R4 fill:#fff8e1,stroke:#f9a825
    style R5 fill:#e3f2fd,stroke:#1565c0
    style R6 fill:#f5f5f5,stroke:#9e9e9e
```

### 9.4 权限检查流程

```mermaid
sequenceDiagram
    participant C as 客户端
    participant API as API Server
    participant Auth as 认证中间件
    participant RBAC as RBAC服务
    participant Store as RBAC存储

    C->>API: 请求API (携带JWT)
    API->>Auth: 验证JWT Token
    Auth-->>API: Token有效，返回用户信息
    API->>RBAC: CheckPermission(userID, requiredPermission)
    RBAC->>Store: GetUserRoles(userID)
    Store-->>RBAC: 返回用户角色列表
    loop 遍历每个角色
        RBAC->>Store: GetRolePermissions(roleID)
        Store-->>RBAC: 返回角色权限列表
    end
    RBAC-->>API: 返回权限检查结果

    alt 有权限
        API->>API: 处理业务逻辑
        API-->>C: 200 OK
    else 无权限
        API-->>C: 403 Forbidden
    end
```

---

## 10. 导出功能设计

### 10.1 导出功能架构

```mermaid
graph TB
    subgraph 导出请求["导出请求层"]
        FE["前端应用"]
        API["导出API"]
    end

    subgraph 导出服务["导出服务层"]
        EM["导出管理器"]
        EF["导出格式化器"]
        WG["Worker Pool<br/>工作池"]
    end

    subgraph 导出器["导出格式化器"]
        JSON["JSON导出器"]
        CSV["CSV导出器"]
        EXCEL["Excel导出器"]
    end

    subgraph 存储["临时存储"]
        FS["文件系统<br/>/export/temp/"]
        REDIS["Redis<br/>任务状态缓存"]
    end

    FE --> API
    API --> EM
    EM --> EF
    EM --> WG
    EF --> JSON
    EF --> CSV
    EF --> EXCEL
    JSON --> FS
    CSV --> FS
    EXCEL --> FS
    EM --> REDIS

    style FE fill:#e3f2fd,stroke:#1565c0
    style API fill:#e8f5e9,stroke:#2e7d32
    style EM fill:#fff8e1,stroke:#f9a825
    style JSON fill:#fce4ec,stroke:#c2185b
    style CSV fill:#fce4ec,stroke:#c2185b
    style EXCEL fill:#fce4ec,stroke:#c2185b
    style FS fill:#f5f5f5,stroke:#9e9e9e
```

### 10.2 导出流程时序图

```mermaid
sequenceDiagram
    participant U as 用户
    participant FE as 前端
    participant API as 导出API
    participant EM as 导出管理器
    participant WF as WorkerPool
    participant EF as 格式化器
    participant FS as 文件系统
    participant DB as MongoDB

    U->>FE: 选择导出选项
    FE->>FE: 选择模块、格式、筛选条件
    FE->>API: POST /api/export/data
    API->>EM: CreateExportTask(params)
    EM->>DB: 查询源数据
    DB-->>EM: 返回数据
    EM-->>API: 返回任务ID
    API-->>FE: 202 Accepted { task_id }
    FE-->>U: 显示导出中

    loop 异步处理
        EM->>WF: SubmitJob(task)
        WF->>EF: FormatData(data, format)
        EF-->>WF: FormattedData
        WF->>FS: SaveFile(data)
        FS-->>WF: FilePath
        WF->>EM: JobComplete(path)
        EM->>DB: 更新任务状态
    end

    U->>FE: 查询导出状态
    FE->>API: GET /api/export/status/:id
    API->>DB: GetTaskStatus
    DB-->>API: status: completed
    API-->>FE: 200 OK { status, download_url }
    FE-->>U: 显示下载按钮

    U->>FE: 点击下载
    FE->>API: GET /api/export/download/:id
    API->>FS: GetFile
    FS-->>API: File
    API-->>FE: File Stream
    FE->>FE: 触发浏览器下载
```

### 10.3 支持的导出格式

```mermaid
graph LR
    subgraph 导出格式["支持的导出格式"]
        JSON["JSON<br/>application/json<br/>结构化数据交换"]
        CSV["CSV<br/>text/csv<br/>表格数据"]
        EXCEL["Excel<br/>application/vnd.openxmlformats<br/>电子表格"]
    end

    subgraph 特性["格式特性"]
        F1["JSON: 嵌套结构<br/>完整数据类型<br/>易于程序处理"]
        F2["CSV: 扁平结构<br/>通用兼容<br/>适合数据分析"]
        F3["Excel: 多Sheet支持<br/>格式保留<br/>适合报表"]
    end

    JSON --> F1
    CSV --> F2
    EXCEL --> F3

    style JSON fill:#fce4ec,stroke:#c2185b
    style CSV fill:#e8f5e9,stroke:#2e7d32
    style EXCEL fill:#e3f2fd,stroke:#1565c0
```

### 10.4 导出API请求响应示例

#### 10.4.1 创建导出任务

**请求:**
```http
POST /api/export/data
Content-Type: application/json
Authorization: Bearer <token>

{
  "module": "movie",
  "format": "csv",
  "jql": "year >= 2020",
  "fields": ["title", "director", "year", "rating"],
  "async": true
}
```

**响应:**
```json
{
  "task_id": "export_123456",
  "status": "pending",
  "estimated_time": 30,
  "created_at": "2026-04-19T10:00:00Z"
}
```

#### 10.4.2 查询导出状态

**请求:**
```http
GET /api/export/status/export_123456
Authorization: Bearer <token>
```

**响应:**
```json
{
  "task_id": "export_123456",
  "status": "completed",
  "progress": 100,
  "file_size": 1024000,
  "download_url": "/api/export/download/export_123456",
  "expires_at": "2026-04-20T10:00:00Z"
}
```

---

## 附录

### A. 索引设计

#### A.1 datacenter 数据库索引

| 集合 | 索引 | 类型 | 说明 |
|------|------|------|------|
| collections | { module: 1 } | Unique | 模块唯一索引 |
| field_definitions | { module: 1, field_name: 1 } | Unique | 复合唯一索引 |
| scrape_tasks | { module: 1, status: 1 } | Compound | 复合索引 |
| scrape_tasks | { created_at: -1 } | Single | 降序索引 |
| scrape_tasks | { business_data_id: 1 } | Single | 关联查询索引 |
| {module}_data | { module: 1 } | Single | 模块索引 |
| {module}_data | { created_at: -1 } | Single | 降序索引 |
| deleted_data | { module: 1 } | Single | 模块索引 |
| deleted_data | { original_id: 1 } | Single | 原始ID索引 |
| deleted_data | { deleted_at: 1 } | Single | 删除时间索引 |

#### A.2 rbac 数据库索引

| 集合 | 索引 | 类型 | 说明 |
|------|------|------|------|
| users | { username: 1 } | Unique | 用户名唯一索引 |
| users | { email: 1 } | Unique | 邮箱唯一索引 |
| permissions | { code: 1 } | Unique | 权限代码唯一索引 |
| roles | { code: 1 } | Unique | 角色代码唯一索引 |

### B. 错误码定义

| 错误码 | HTTP状态码 | 说明 |
|--------|------------|------|
| AUTH_001 | 401 | 无效的凭据 |
| AUTH_002 | 401 | Token已过期 |
| AUTH_003 | 401 | 缺少认证头 |
| PERM_001 | 403 | 无访问权限 |
| USER_001 | 404 | 用户不存在 |
| USER_002 | 400 | 用户名已存在 |
| ROLE_001 | 404 | 角色不存在 |
| DATA_001 | 404 | 数据不存在 |
| DATA_002 | 400 | 数据验证失败 |
| SCRAPE_001 | 400 | 刮削任务创建失败 |
| SCRAPE_002 | 404 | 刮削任务不存在 |
| EXPORT_001 | 400 | 导出格式不支持 |
| EXPORT_002 | 404 | 导出任务不存在 |
| EXPORT_003 | 410 | 导出文件已过期 |

---

*文档结束*

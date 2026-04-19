# 数据中心系统需求设计文档

## 1. 系统概述

### 1.1 项目背景

数据中心系统是一个基于Go和MongoDB构建的企业级数据管理平台，提供用户权限管理、业务数据管理、数据刮削等核心功能。系统采用前后端分离架构，前端使用React+TypeScript，后端使用Gin框架。

### 1.2 系统目标

- 实现基于RBAC的权限管理系统，支持用户、角色、权限的完整生命周期管理
- 提供灵活的业务数据管理能力，支持动态字段定义和数据模型扩展
- 实现异步数据刮削系统，支持高并发任务处理
- 提供RESTful API，支持前端SPA应用访问
- 实现完整的日志记录和审计功能

### 1.3 适用范围

- 企业内部数据管理系统
- 需要权限控制的后台管理系统
- 数据采集和整合平台

## 2. 功能需求

### 2.1 认证与授权

#### 2.1.1 用户登录

| 功能 | 描述 |
|------|------|
| 用户名密码认证 | 支持用户名+密码登录，使用bcrypt密码哈希 |
| JWT Token | 登录成功后返回JWT Token，有效期24小时 |
| Token刷新 | 支持Token刷新机制，刷新Token有效期30天 |

#### 2.1.2 认证流程

```
用户登录 → 验证用户名密码 → 生成JWT Token → 返回Token给客户端
    ↓
客户端请求 →携带Token → 中间件验证Token → 验证通过 → 处理请求
```

### 2.2 用户管理

#### 2.2.1 功能列表

| 功能 | 描述 |
|------|------|
| 用户CRUD | 创建、读取、更新、删除用户 |
| 用户列表 | 分页获取用户列表，支持关键词搜索 |
| 角色分配 | 为用户分配/移除角色，支持多角色 |
| 用户详情 | 获取用户详细信息及关联角色 |

#### 2.2.2 用户属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名，唯一 |
| email | string | 是 | 邮箱，唯一 |
| password | string | 是 | 密码，bcrypt加密 |
| phone | string | 否 | 电话号码 |
| address | string | 否 | 地址 |
| role_ids | string[] | 否 | 关联角色ID数组 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

#### 2.2.3 验证规则

- 用户名唯一，长度3-50字符
- 邮箱格式正确，唯一
- 密码至少8位

### 2.3 角色管理

#### 2.3.1 功能列表

| 功能 | 描述 |
|------|------|
| 角色CRUD | 创建、读取、更新、删除角色 |
| 角色列表 | 分页获取角色列表 |
| 权限分配 | 为角色分配/移除权限 |
| 角色详情 | 获取角色详细信息及关联权限 |

#### 2.3.2 角色属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 角色名称 |
| code | string | 是 | 角色代码，唯一 |
| description | string | 否 | 角色描述 |
| permission_ids | string[] | 否 | 关联权限ID数组 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

#### 2.3.3 内置角色

| 角色代码 | 角色名称 | 描述 |
|----------|----------|------|
| admin | 超级管理员 | 拥有所有权限 |
| user | 普通用户 | 基础操作权限 |
| read_only | 只读用户 | 仅查看权限 |

### 2.4 权限管理

#### 2.4.1 功能列表

| 功能 | 描述 |
|------|------|
| 权限CRUD | 创建、读取、更新、删除权限 |
| 权限列表 | 分页获取权限列表，支持模块筛选 |
| 权限详情 | 获取权限详细信息 |

#### 2.4.2 权限属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 权限名称 |
| code | string | 是 | 权限代码，格式：模块:操作 |
| description | string | 否 | 权限描述 |
| module | string | 否 | 所属模块 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

#### 2.4.3 内置权限

| 权限代码 | 权限名称 | 模块 |
|----------|----------|------|
| user:* | 用户完全控制 | user |
| user:read | 查看用户 | user |
| user:write | 管理用户 | user |
| role:* | 角色完全控制 | role |
| role:read | 查看角色 | role |
| role:write | 管理角色 | role |
| permission:* | 权限完全控制 | permission |
| permission:read | 查看权限 | permission |
| permission:write | 管理权限 | permission |
| data:* | 数据完全控制 | data |
| data:read | 查看数据 | data |
| data:write | 管理数据 | data |

### 2.5 业务数据管理

#### 2.5.1 功能列表

| 功能 | 描述 |
|------|------|
| 字段定义CRUD | 定义各模块的自定义字段 |
| 集合管理 | 创建、更新、删除数据集合 |
| 索引管理 | 创建、删除集合索引 |
| 业务数据CRUD | 各模块业务数据的增删改查 |
| 软删除 | 支持数据软删除，可恢复 |
| JQL查询 | 支持类JQL语法查询数据 |

#### 2.5.2 字段定义属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 所属模块 |
| name | string | 是 | 字段名称 |
| code | string | 是 | 字段代码 |
| type | string | 是 | 字段类型(string/number/date/boolean) |
| required | boolean | 否 | 是否必填 |
| default_value | any | 否 | 默认值 |
| description | string | 否 | 字段描述 |

#### 2.5.3 集合属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 集合名称 |
| code | string | 是 | 集合代码 |
| description | string | 否 | 集合描述 |
| fields | object | 否 | 字段定义 |

#### 2.5.4 业务数据属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 所属模块 |
| data | object | 是 | 业务数据 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

### 2.6 数据刮削系统

#### 2.6.1 功能列表

| 功能 | 描述 |
|------|------|
| 任务提交 | 提交数据刮削任务到队列 |
| 异步处理 | 工作协程池异步处理任务 |
| 任务状态 | 支持pending/scraping/success/failed状态 |
| 任务重试 | 失败任务支持重试 |
| 任务软删除 | 删除任务移动到deleted_scrape_tasks集合 |
| 任务恢复 | 从deleted_scrape_tasks恢复任务 |

#### 2.6.2 刮削任务属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | string | 是 | 任务ID |
| module | string | 是 | 所属模块 |
| url | string | 是 | 刮削URL |
| status | string | 否 | 任务状态 |
| retry_count | int | 否 | 重试次数 |
| max_retries | int | 否 | 最大重试次数 |
| error_message | string | 否 | 错误信息 |
| created_at | datetime | 否 | 创建时间 |
| started_at | datetime | 否 | 开始时间 |
| finished_at | datetime | 否 | 完成时间 |

#### 2.6.3 任务状态流

```
pending → scraping → success
              ↓
            failed → (重试) → scraping
```

### 2.7 日志系统

#### 2.7.1 日志类型

| 日志类型 | 文件 | 说明 |
|----------|------|------|
| HTTP日志 | logs/http.log | API请求响应记录 |
| 应用日志 | logs/app.log | 程序运行日志 |
| 刮削日志 | logs/scraper.log | 刮削任务执行记录 |

#### 2.7.2 日志格式

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

## 3. 非功能需求

### 3.1 性能需求

| 指标 | 要求 |
|------|------|
| API响应时间 | P95 < 200ms |
| 并发用户数 | 支持100+并发用户 |
| 刮削吞吐量 | 4工作协程，每秒处理10+任务 |

### 3.2 安全需求

| 项目 | 要求 |
|------|------|
| 密码加密 | bcrypt，cost factor 10 |
| JWT安全 | HS256签名，24小时有效期 |
| 输入验证 | 所有输入参数验证 |
| SQL注入 | 防止NoSQL注入 |
| CORS | 支持跨域配置 |

### 3.3 可靠性需求

| 项目 | 要求 |
|------|------|
| 日志轮转 | 单文件100MB，保留5个备份 |
| 错误处理 | 统一错误响应格式 |
| 软删除 | 数据可恢复 |
| 任务重试 | 失败任务自动重试3次 |

### 3.4 可维护性需求

| 项目 | 要求 |
|------|------|
| 代码结构 | 分层架构，模块化 |
| 注释 | 公共接口注释 |
| 配置 | 环境变量配置 |
| 文档 | API文档 |

## 4. 接口规范

### 4.1 通用响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

### 4.2 错误响应格式

```json
{
  "code": 400,
  "message": "error message"
}
```

### 4.3 HTTP状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |

## 5. 数据字典

### 5.1 用户状态

| 状态码 | 状态名称 | 说明 |
|--------|----------|------|
| active | 活跃 | 正常使用的用户 |
| inactive | 非活跃 | 被禁用的用户 |

### 5.2 任务状态

| 状态码 | 状态名称 | 说明 |
|--------|----------|------|
| pending | 等待中 | 任务等待处理 |
| scraping | 处理中 | 任务正在处理 |
| success | 成功 | 任务成功完成 |
| failed | 失败 | 任务处理失败 |

### 5.3 权限模块

| 模块代码 | 模块名称 | 说明 |
|----------|----------|------|
| user | 用户管理 | 用户相关权限 |
| role | 角色管理 | 角色相关权限 |
| permission | 权限管理 | 权限相关权限 |
| data | 数据管理 | 数据相关权限 |

### 5.4 数据库集合

#### datacenter 数据库

| 集合名 | 说明 |
|--------|------|
| collections | 集合元数据 |
| field_definitions | 字段定义 |
| scrape_tasks | 刮削任务 |
| deleted_scrape_tasks | 已删除刮削任务(软删除) |
| {module}_data | 各模块业务数据(动态) |
| deleted_data | 已删除业务数据(软删除) |

#### rbac 数据库

| 集合名 | 说明 |
|--------|------|
| users | 用户信息 |
| permissions | 权限定义 |
| roles | 角色定义 |

## 6. 业务流程

### 6.1 用户登录流程

```
1. 用户输入用户名密码
2. 前端发送POST /api/auth/login
3. 后端验证用户名密码
4. 验证成功，生成JWT Token
5. 返回Token给前端
6. 前端存储Token到localStorage
7. 后续请求携带Token
```

### 6.2 权限检查流程

```
1. 用户请求受保护资源
2. 中间件提取Token
3. 验证Token有效性
4. 解析用户信息和权限
5. 检查所需权限
6. 有权限则放行，无权限返回403
```

### 6.3 数据刮削流程

```
1. 用户提交业务数据
2. 系统保存数据到MongoDB
3. 创建刮削任务，状态为pending
4. 任务放入队列
5. 工作协程取出任务
6. 更新任务状态为scraping
7. 执行刮削逻辑
8. 更新任务状态为success/failed
9. 如失败且未超过最大重试次数，重新入队
```

### 6.4 数据删除/恢复流程

```
1. 用户发起删除请求
2. 系统将数据移动到deleted_data或deleted_scrape_tasks集合
3. 保留原始数据的所有信息
4. 用户可查询已删除数据列表
5. 用户发起恢复请求
6. 系统将数据从deleted集合恢复到原始集合
7. 删除deleted集合中的记录
```

### 6.5 刮削任务删除/恢复流程

```
1. 用户发起刮削任务删除请求
2. 系统将任务移动到deleted_scrape_tasks集合
3. 保留原始任务的所有信息
4. 用户可查询已删除任务列表
5. 用户发起恢复请求
6. 系统将任务恢复到scrape_tasks集合
7. 删除deleted_scrape_tasks中的记录
```

## 7. 风险与约束

### 7.1 技术约束

- Go 1.20+
- MongoDB 6.0+
- Node.js 18+ (前端)
- React 18+ (前端)

### 7.2 已知风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| MongoDB连接失败 | 系统不可用 | 连接池配置，重试机制 |
| JWT Token泄露 | 安全风险 | HTTPS传输，Token加密 |
| 刮削任务堆积 | 数据延迟 | 工作协程池，队列大小限制 |

## 8. 术语表

| 术语 | 说明 |
|------|------|
| RBAC | Role-Based Access Control，基于角色的访问控制 |
| JWT | JSON Web Token，JSON格式的Token |
| JQL | JIRA Query Language，查询语法 |
| 刮削 | Web数据抓取 |
| 软删除 | 标记删除而非物理删除 |
| MongoDB | 文档型数据库 |

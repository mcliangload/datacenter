# 数据中心系统需求设计文档

## 1. 系统概述

### 1.1 项目背景

数据中心系统是一个基于Go和MongoDB构建的企业级数据管理平台，提供用户权限管理、业务数据管理、数据刮削等核心功能。系统采用前后端分离架构，前端使用React+TypeScript，后端使用Gin框架。

### 1.2 系统目标

- 实现基于RBAC的权限管理系统，支持用户、角色、权限的完整生命周期管理
- 提供灵活的业务数据管理能力，支持动态字段定义和数据模型扩展
- 实现异步数据刮削系统，支持高并发任务处理
- 提供RESTful API，支持前端SPA应用访问
- 实现完整的日志记录功能

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
| JWT Token | 登录成功后返回JWT Token，默认有效期24小时 |
| Token刷新 | 支持Token刷新机制，默认刷新有效期720小时(30天) |
| 用户注册 | 支持新用户注册 |

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
| 用户列表 | 分页获取用户列表 |
| 角色分配 | 为用户分配/移除角色，支持多角色 |
| 用户详情 | 获取用户详细信息及关联角色 |

#### 2.2.2 用户属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名，唯一 |
| email | string | 是 | 邮箱，唯一 |
| password | string | 是 | 密码，bcrypt加密 |
| role_ids | string[] | 否 | 关联角色ID数组 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

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

### 2.4 权限管理

#### 2.4.1 功能列表

| 功能 | 描述 |
|------|------|
| 权限CRUD | 创建、读取、更新、删除权限 |
| 权限列表 | 分页获取权限列表 |
| 权限详情 | 获取权限详细信息 |

#### 2.4.2 权限属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 权限名称 |
| code | string | 是 | 权限代码，格式：模块:操作 |
| description | string | 否 | 权限描述 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

#### 2.4.3 权限代码格式

系统使用基于资源的权限代码格式，支持通配符：

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
| 集合RBAC | 每个集合独立的基于角色的访问控制系统 |

#### 2.5.2 字段定义属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 所属模块 |
| field_name | string | 是 | 字段名称 |
| field_type | string | 是 | 字段类型(string/number/boolean/date/array/object) |
| description | string | 否 | 字段描述 |
| required | boolean | 否 | 是否必填 |
| default_value | any | 否 | 默认值 |
| constraints | object | 否 | 字段约束 |

#### 2.5.3 集合属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 模块名称 |
| description | string | 否 | 集合描述 |
| datatype_owner | string | 否 | 数据类型所有者 |
| collection_name | string | 是 | MongoDB集合名称 |

#### 2.5.4 业务数据属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 所属模块 |
| description | string | 否 | 描述信息 |
| custom_fields | object | 否 | 自定义字段 |
| file_path | string | 否 | 文件路径 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

### 2.6 集合特定RBAC系统

#### 2.6.1 功能列表

| 功能 | 描述 |
|------|------|
| 自动角色创建 | 创建新集合时自动创建管理员、操作员、普通用户三种角色 |
| 集合角色分配 | 为用户分配集合级别的角色 |
| 权限检查 | 集合数据访问时进行权限验证 |
| 审计日志 | 记录角色分配和权限变更操作 |
| 超级管理员权限 | 拥有system:admin权限的用户可以访问所有集合 |

#### 2.6.2 集合角色类型

| 角色类型 | 代码 | 权限 |
|----------|------|------|
| 集合管理员 | admin | collection:admin, collection:read, collection:write, collection:delete, collection:field:admin |
| 集合操作员 | operator | collection:read, collection:write, collection:delete |
| 集合普通用户 | user | collection:read |

#### 2.6.3 集合权限代码格式

| 权限代码 | 描述 |
|----------|------|
| collection:admin | 集合管理权限（包含所有其他集合权限） |
| collection:read | 查看集合数据 |
| collection:write | 创建和更新集合数据 |
| collection:delete | 删除集合数据 |
| collection:field:admin | 管理集合字段定义 |

#### 2.6.4 集合角色属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| collection_module | string | 是 | 所属集合模块 |
| name | string | 是 | 角色名称 |
| code | string | 是 | 角色代码 |
| type | string | 是 | 角色类型(admin/operator/user) |
| description | string | 否 | 角色描述 |
| permission_ids | string[] | 否 | 关联权限ID数组 |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

#### 2.6.5 集合角色分配属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | string | 是 | 用户ID |
| collection_module | string | 是 | 集合模块 |
| collection_role_id | string | 是 | 集合角色ID |
| created_by | string | 否 | 创建人 |
| created_at | datetime | 否 | 创建时间 |
| updated_by | string | 否 | 更新人 |
| updated_at | datetime | 否 | 更新时间 |

#### 2.6.6 集合角色分配规则

- 集合管理员角色只能由超级管理员分配
- 每个集合只能有一个管理员
- 集合操作员角色由管理员或超级管理员分配
- 集合普通用户角色由管理员或超级管理员分配

#### 2.6.7 审计日志属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | string | 是 | 操作者用户ID |
| username | string | 是 | 操作者用户名 |
| action | string | 是 | 操作类型 |
| resource | string | 是 | 资源类型 |
| resource_id | string | 是 | 资源ID |
| details | string | 否 | 操作详情 |
| ip_address | string | 否 | IP地址 |
| user_agent | string | 否 | 用户代理 |
| timestamp | datetime | 否 | 操作时间 |

#### 2.6.8 集合RBAC API接口

| 接口 | 方法 | 描述 |
|------|------|------|
| /api/collections/{module}/roles | GET | 获取集合的所有角色 |
| /api/collections/{module}/roles/assignments | GET | 获取集合的角色分配列表 |
| /api/collections/{module}/roles/assign | POST | 为用户分配集合角色 |
| /api/collections/{module}/roles/{roleId}/assignments/{userId} | DELETE | 移除用户的集合角色 |
| /api/collections/{module}/audit-logs | GET | 获取集合的审计日志 |
| /api/collection-data/module/{module}/* | * | 集合数据访问接口（需集合权限） |

#### 2.6.9 集合数据访问权限检查流程

```
1. 用户请求集合数据接口（如 GET /api/collection-data/module/test_collection）
2. 中间件提取用户ID
3. 检查用户是否拥有system:admin全局权限
   - 如果有，则允许访问
4. 检查用户在集合中的角色分配
5. 检查用户角色是否具有所需权限
6. 有权限则放行，无权限返回403
```

### 2.7 数据刮削系统

#### 2.6.1 功能列表

| 功能 | 描述 |
|------|------|
| 任务提交 | 提交数据刮削任务到队列 |
| 异步处理 | 工作协程池异步处理任务 |
| 任务状态 | 支持pending/scraping/success/failed状态 |
| 任务重试 | 失败任务支持重试 |
| 任务软删除 | 删除任务移动到deleted_scrape_tasks集合 |
| 任务恢复 | 从deleted_scrape_tasks恢复任务 |
| 批量删除 | 支持批量删除刮削任务 |

#### 2.6.2 刮削任务属性

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 所属模块 |
| data_path | string | 是 | 数据文件路径 |
| scraper_path | string | 是 | 刮削器脚本路径 |
| status | string | 否 | 任务状态 |
| result | object | 否 | 刮削结果 |
| error_message | string | 否 | 错误信息 |
| started_at | datetime | 否 | 开始时间 |
| completed_at | datetime | 否 | 完成时间 |
| business_data_id | string | 否 | 关联的业务数据ID |
| description | string | 否 | 任务描述 |

#### 2.6.3 任务状态流

```
pending → scraping → success
              ↓
            failed → (重试) → pending
```

### 2.7 日志系统

#### 2.7.1 日志类型

| 日志类型 | 文件 | 说明 |
|----------|------|------|
| HTTP日志 | logs/http.log | API请求响应记录 |
| 应用日志 | logs/app.log | 程序运行日志 |

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
| 密码加密 | bcrypt加密 |
| JWT安全 | HS256签名，默认24小时有效期 |
| 输入验证 | 所有输入参数验证 |
| CORS | 支持跨域配置 |

### 3.3 可靠性需求

| 项目 | 要求 |
|------|------|
| 日志轮转 | 单文件100MB，保留5个备份，30天保留 |
| 错误处理 | 统一错误响应格式 |
| 软删除 | 数据可恢复 |
| 任务队列 | 队列大小1000 |

### 3.4 可维护性需求

| 项目 | 要求 |
|------|------|
| 代码结构 | 分层架构，模块化 |
| 配置 | 环境变量配置 |
| 文档 | API文档 |

## 4. 接口规范

### 4.1 通用响应格式

```json
{
  "data": {...},
  "message": "success"
}
```

### 4.2 错误响应格式

```json
{
  "error": "error message"
}
```

### 4.3 HTTP状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |

## 5. 数据字典

### 5.1 任务状态

| 状态码 | 状态名称 | 说明 |
|--------|----------|------|
| pending | 等待中 | 任务等待处理 |
| scraping | 处理中 | 任务正在处理 |
| success | 成功 | 任务成功完成 |
| failed | 失败 | 任务处理失败 |

### 5.2 字段类型

| 类型代码 | 类型名称 | 说明 |
|----------|----------|------|
| string | 字符串 | 字符串类型 |
| number | 数字 | 数字类型 |
| boolean | 布尔 | 布尔类型 |
| date | 日期 | 日期类型 |
| array | 数组 | 数组类型 |
| object | 对象 | 对象类型 |

### 5.3 数据库集合

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
6. 前端存储Token
7. 后续请求携带Token
```

### 6.2 权限检查流程

```
1. 用户请求受保护资源
2. 中间件提取Token
3. 验证Token有效性
4. 解析用户信息和权限
5. 检查所需权限（支持通配符匹配）
6. 有权限则放行，无权限返回403
```

### 6.3 数据刮削流程

```
1. 用户提交刮削任务
2. 创建刮削任务，状态为pending
3. 任务放入队列
4. 工作协程取出任务
5. 更新任务状态为scraping
6. 执行刮削器脚本
7. 更新任务状态为success/failed
8. 成功时存储结果到业务数据集合
```

### 6.4 数据删除/恢复流程

```
1. 用户发起删除请求
2. 系统将数据移动到deleted_data集合
3. 保留原始数据的所有信息
4. 用户可查询已删除数据列表
5. 用户发起恢复请求
6. 系统将数据从deleted集合恢复到原始集合
7. 删除deleted集合中的记录
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

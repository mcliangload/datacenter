# 数据中心系统 API 文档

## 概述

数据中心系统提供完整的 RESTful API，包括用户认证、RBAC权限管理、业务数据管理和数据刮削功能。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **认证方式**: JWT Bearer Token
- **Content-Type**: `application/json`

## 认证

### 登录

```
POST /api/auth/login
```

**请求体**:

```json
{
  "username": "admin",
  "password": "liangminchuan"
}
```

**响应**:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "69e4976f843426c5b9c31f9f",
    "username": "admin",
    "email": "admin@datacenter.local",
    "roles": ["role_id_1", "role_id_2"]
  }
}
```

### 注册

```
POST /api/auth/register
```

**请求体**:

```json
{
  "username": "newuser",
  "password": "password123",
  "email": "newuser@example.com"
}
```

**响应**: `201 Created`

```json
{
  "id": "new_user_id",
  "username": "newuser",
  "email": "newuser@example.com"
}
```

***

## 用户管理

### 获取用户列表

```
GET /api/users?page=1&pageSize=10
```

**参数**:

| 参数       | 类型  | 默认值 | 描述   |
| -------- | --- | --- | ---- |
| page     | int | 1   | 当前页码 |
| pageSize | int | 10  | 每页数量 |

**响应**:

```json
{
  "data": [
    {
      "_id": "69e4976f843426c5b9c31f9f",
      "username": "admin",
      "email": "admin@datacenter.local",
      "role_ids": ["role_id_1", "role_id_2"],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "pageSize": 10
}
```

### 获取单个用户

```
GET /api/users/:id
```

**响应**:

```json
{
  "_id": "69e4976f843426c5b9c31f9f",
  "username": "admin",
  "email": "admin@datacenter.local",
  "role_ids": ["role_id_1"],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### 创建用户

```
POST /api/users
```

**请求体**:

```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "password123",
  "role_ids": ["role_id_1"]
}
```

**响应**: `201 Created`

```json
{
  "_id": "new_user_id",
  "username": "newuser",
  "email": "newuser@example.com",
  "role_ids": ["role_id_1"],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### 更新用户

```
PUT /api/users/:id
```

**请求体**:

```json
{
  "email": "updated@example.com",
  "password": "newpassword",
  "role_ids": ["role_id_1", "role_id_2"]
}
```

### 删除用户

```
DELETE /api/users/:id
```

### 分配角色给用户

```
POST /api/users/:id/roles
```

**请求体**:

```json
{
  "role_id": "role_id"
}
```

### 从用户移除角色

```
DELETE /api/users/:id/roles/:roleId
```

### 获取用户角色

```
GET /api/users/:id/roles
```

***

## 权限管理

### 获取权限列表

```
GET /api/permissions?page=1&pageSize=10
```

**参数**:

| 参数       | 类型  | 默认值 | 描述   |
| -------- | --- | --- | ---- |
| page     | int | 1   | 当前页码 |
| pageSize | int | 10  | 每页数量 |

**响应**:

```json
{
  "data": [
    {
      "_id": "permission_id",
      "name": "用户管理",
      "code": "user:write",
      "description": "管理系统用户账户"
    }
  ],
  "total": 20,
  "page": 1,
  "pageSize": 10
}
```

### 获取单个权限

```
GET /api/permissions/:id
```

### 创建权限

```
POST /api/permissions
```

**请求体**:

```json
{
  "name": "数据管理",
  "code": "data:write",
  "description": "管理系统数据"
}
```

### 更新权限

```
PUT /api/permissions/:id
```

### 删除权限

```
DELETE /api/permissions/:id
```

***

## 角色管理

### 获取角色列表

```
GET /api/roles?page=1&pageSize=10
```

**参数**:

| 参数       | 类型  | 默认值 | 描述   |
| -------- | --- | --- | ---- |
| page     | int | 1   | 当前页码 |
| pageSize | int | 10  | 每页数量 |

**响应**:

```json
{
  "data": [
    {
      "_id": "role_id",
      "name": "管理员",
      "code": "admin",
      "description": "系统管理员",
      "permission_ids": ["perm_id_1", "perm_id_2"]
    }
  ],
  "total": 10,
  "page": 1,
  "pageSize": 10
}
```

### 获取单个角色

```
GET /api/roles/:id
```

### 创建角色

```
POST /api/roles
```

**请求体**:

```json
{
  "name": "数据管理员",
  "code": "data_admin",
  "description": "管理系统数据",
  "permission_ids": ["permission_id_1"]
}
```

### 更新角色

```
PUT /api/roles/:id
```

### 删除角色

```
DELETE /api/roles/:id
```

### 分配权限给角色

```
POST /api/roles/:id/permissions
```

**请求体**:

```json
{
  "permission_id": "permission_id"
}
```

### 从角色移除权限

```
DELETE /api/roles/:id/permissions/:permissionId
```

### 获取角色权限

```
GET /api/roles/:id/permissions
```

***

## 业务数据管理

### 获取业务数据列表

```
GET /api/business/module/:module?page=1&pageSize=10&jql=field:value
```

**参数**:

| 参数       | 类型     | 默认值 | 描述   |
| -------- | ------ | --- | ---- |
| page     | int    | 1   | 当前页码 |
| pageSize | int    | 10  | 每页数量 |
| jql      | string | -   | 过滤条件 |

**响应**:

```json
{
  "data": [
    {
      "_id": "data_id",
      "module": "movie",
      "description": "电影数据",
      "custom_fields": {
        "title": "电影标题",
        "director": "导演",
        "year": 2024
      },
      "file_path": "/data/movie.json",
      "created_by": "admin",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_by": "admin",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "pageSize": 10
}
```

### 获取单个业务数据

```
GET /api/business/module/:module/:id
```

### 创建业务数据

```
POST /api/business
```

**请求体**:

```json
{
  "module": "movie",
  "data": {
    "title": "电影标题",
    "director": "导演",
    "year": 2024
  },
  "description": "电影数据描述"
}
```

**响应**:

```json
{
  "message": "数据创建成功",
  "data": {
    "_id": "data_id",
    "module": "movie",
    "description": "电影数据描述",
    "custom_fields": {
      "title": "电影标题",
      "director": "导演",
      "year": 2024
    },
    "created_by": "admin",
    "created_at": "2024-01-15T10:30:00Z"
  },
  "module": "movie"
}
```

**注意**: 如果模块不存在，会返回错误:

```json
{
  "error": "模块不存在: movie，请先创建模块集合"
}
```

### 更新业务数据

```
PUT /api/business/module/:module/:id
```

**请求体**:

```json
{
  "description": "更新后的描述",
  "data": {
    "title": "更新后的标题",
    "year": 2025
  }
}
```

### 删除业务数据

```
DELETE /api/business/module/:module/:id
```

***

## 集合管理

### 获取集合列表

```
GET /api/collections?page=1&pageSize=10
```

**响应**:

```json
{
  "data": [
    {
      "_id": "collection_id",
      "module": "movie",
      "description": "电影数据模块",
      "datatype_owner": "admin",
      "collection_name": "movie_data",
      "created_by": "admin",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 10,
  "total": 5
}
```

### 获取单个集合

```
GET /api/collections/:module
```

### 创建集合

```
POST /api/collections
```

**请求体**:

```json
{
  "module": "movie",
  "description": "电影数据模块",
  "datatype_owner": "admin"
}
```

### 更新集合

```
PUT /api/collections/:module
```

**请求体**:

```json
{
  "description": "更新后的描述",
  "datatype_owner": "new_owner"
}
```

### 删除集合

```
DELETE /api/collections/:module
```

### 创建索引

```
POST /api/collections/:module/indexes
```

**请求体**:

```json
{
  "keys": { "field_name": 1 },
  "options": {
    "unique": false,
    "background": true,
    "name": "field_name_index"
  }
}
```

### 获取索引列表

```
GET /api/collections/:module/indexes
```

### 删除索引

```
DELETE /api/collections/:module/indexes/:name
```

***

## 刮削任务管理

### 提交刮削任务

```
POST /api/scraper/upload
```

**请求体**:

```json
{
  "module": "movie",
  "data_path": "/data/movies.json",
  "scraper_path": "/scrapers/movie_scraper.py",
  "description": "刮削任务描述"
}
```

**响应**:

```json
{
  "message": "Scrape task submitted successfully",
  "task_id": "task_id"
}
```

### 获取刮削任务列表

```
GET /api/scraper/tasks?module=movie&status=success&page=1&pageSize=10
```

**参数**:

| 参数       | 类型     | 描述                             |
| -------- | ------ | ------------------------------ |
| module   | string | 模块名                            |
| status   | string | 任务状态 (pending/scraping/success/failed) |
| page     | int    | 当前页码                           |
| pageSize | int    | 每页数量                           |

### 获取单个刮削任务

```
GET /api/scraper/tasks/:id
```

### 重试刮削任务

```
POST /api/scraper/tasks/:id/retry
```

**请求体**:

```json
{
  "scraper_path": "/scrapers/movie_scraper.py"
}
```

### 删除刮削任务

```
DELETE /api/scraper/tasks/:id
```

### 批量删除刮削任务

```
POST /api/scraper/tasks/batch-delete
```

**请求体**:

```json
{
  "ids": ["task_id_1", "task_id_2"]
}
```

***

## 已删除数据管理

### 获取已删除数据列表

```
GET /api/deleted/module/:module?page=1&pageSize=10
```

### 获取单个已删除数据

```
GET /api/deleted/:id
```

### 恢复已删除数据

```
POST /api/deleted/:id/recover
```

***

## 已删除刮削任务管理

### 获取已删除刮削任务列表

```
GET /api/deleted-scraper/module/:module?page=1&pageSize=10
```

**注意**: 当 module 参数为 "all" 时，查询所有模块的已删除任务

### 获取单个已删除刮削任务

```
GET /api/deleted-scraper/:id
```

### 恢复已删除刮削任务

```
POST /api/deleted-scraper/:id/recover
```

***

## 字段定义管理

### 创建字段定义

```
POST /api/fields
```

**请求体**:

```json
{
  "module": "movie",
  "field_name": "title",
  "field_type": "string",
  "description": "电影标题",
  "constraints": {
    "type": "string",
    "min_length": 1,
    "max_length": 200
  }
}
```

### 获取模块的字段定义

```
GET /api/fields/module/:module
```

### 获取单个字段定义

```
GET /api/fields/:id
```

### 更新字段定义

```
PUT /api/fields/:id
```

**请求体**:

```json
{
  "field_name": "updated_title",
  "field_type": "string",
  "description": "更新后的描述",
  "constraints": {
    "type": "string",
    "min_length": 1,
    "max_length": 100
  }
}
```

### 删除字段定义

```
DELETE /api/fields/:id
```

***

## 错误响应

所有错误响应都遵循以下格式:

```json
{
  "error": "错误描述信息"
}
```

常见状态码:

- `200 OK` - 请求成功
- `201 Created` - 资源创建成功
- `400 Bad Request` - 请求参数错误
- `401 Unauthorized` - 未授权/Token无效
- `403 Forbidden` - 无权限
- `404 Not Found` - 资源不存在
- `500 Internal Server Error` - 服务器内部错误

***

## 分页响应格式

所有列表查询接口都返回分页响应，格式如下:

```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "pageSize": 10
}
```

其中:

- `data`: 数据列表
- `total`: 数据总量（用于计算总页数）
- `page`: 当前页码
- `pageSize`: 每页数量

**总页数计算**: `totalPages = Math.ceil(total / pageSize)`

***

## 权限说明

系统使用基于资源的权限代码格式，支持通配符匹配：

| 权限代码 | 描述 |
|------|------|
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

**通配符匹配规则**: 例如 `user:*` 权限可以匹配 `user:read`、`user:write` 等所有 user 模块的权限。

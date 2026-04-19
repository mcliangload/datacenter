# 数据中心系统API规格文档

## 1. API概述

本文档详细描述数据中心系统的RESTful API接口，包括认证、用户管理、角色管理、权限管理、业务数据管理等模块。

### 1.1 基础信息

| 项目    | 值                           |
| ----- | --------------------------- |
| 基础URL | <http://localhost:8080/api> |
| 数据格式  | JSON                        |
| 字符编码  | UTF-8                       |
| 认证方式  | Bearer Token (JWT)          |

### 1.2 通用请求头

| 头信息           | 必填 | 说明                      |
| ------------- | -- | ----------------------- |
| Content-Type  | 是  | application/json        |
| Authorization | 否  | Bearer {token}，受保护接口需携带 |

### 1.3 通用响应格式

#### 成功响应

```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

#### 分页响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "pageSize": 10
  }
}
```

#### 错误响应

```json
{
  "code": 400,
  "message": "error message"
}
```

## 2. 认证接口

### 2.1 用户登录

**请求**

```
POST /api/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "507f1f77bcf86cd799439011",
      "username": "admin",
      "email": "admin@example.com",
      "roles": ["admin"]
    }
  }
}
```

**错误响应**

| 状态码 | code | 场景       |
| --- | ---- | -------- |
| 401 | 401  | 用户名或密码错误 |

**示例**

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

## 3. 用户管理接口

### 3.1 创建用户

**请求**

```
POST /api/users
Authorization: Bearer {token}
Content-Type: application/json

{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "password123",
  "phone": "13800138000",
  "address": "北京市朝阳区"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439012",
    "username": "newuser",
    "email": "newuser@example.com",
    "phone": "13800138000",
    "address": "北京市朝阳区",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

**错误响应**

| 状态码 | code  | 场景      |
| --- | ----- | ------- |
| 400 | 40001 | 用户名已存在  |
| 400 | 40002 | 邮箱已存在   |
| 400 | 40003 | 密码格式不正确 |

### 3.2 获取用户列表

**请求**

```
GET /api/users?page=1&pageSize=10&keyword=admin
Authorization: Bearer {token}
```

**Query参数**

| 参数       | 类型     | 必填 | 说明        |
| -------- | ------ | -- | --------- |
| page     | int    | 否  | 页码，默认1    |
| pageSize | int    | 否  | 每页数量，默认10 |
| keyword  | string | 否  | 搜索关键词     |

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "507f1f77bcf86cd799439011",
        "username": "admin",
        "email": "admin@example.com",
        "phone": "13800138000",
        "roles": ["admin"]
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
}
```

### 3.3 获取用户详情

**请求**

```
GET /api/users/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "username": "admin",
    "email": "admin@example.com",
    "phone": "13800138000",
    "address": "北京市海淀区",
    "roles": [
      {
        "id": "507f1f77bcf86cd799439021",
        "name": "超级管理员",
        "code": "admin"
      }
    ],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 3.4 更新用户

**请求**

```
PUT /api/users/:id
Authorization: Bearer {token}
Content-Type: application/json

{
  "email": "newemail@example.com",
  "phone": "13900139000",
  "address": "上海市浦东新区"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "username": "admin",
    "email": "newemail@example.com",
    "phone": "13900139000",
    "address": "上海市浦东新区",
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

### 3.5 删除用户

**请求**

```
DELETE /api/users/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

### 3.6 分配角色

**请求**

```
POST /api/users/:id/roles
Authorization: Bearer {token}
Content-Type: application/json

{
  "role_ids": ["507f1f77bcf86cd799439021", "507f1f77bcf86cd799439022"]
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "username": "admin",
    "roles": [
      {"id": "507f1f77bcf86cd799439021", "name": "超级管理员", "code": "admin"},
      {"id": "507f1f77bcf86cd799439022", "name": "普通用户", "code": "user"}
    ]
  }
}
```

### 3.7 移除角色

**请求**

```
DELETE /api/users/:id/roles/:roleId
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

### 3.8 获取用户角色

**请求**

```
GET /api/users/:id/roles
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": "507f1f77bcf86cd799439021",
      "name": "超级管理员",
      "code": "admin",
      "description": "拥有所有权限"
    }
  ]
}
```

## 4. 角色管理接口

### 4.1 创建角色

**请求**

```
POST /api/roles
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "数据管理员",
  "code": "data_admin",
  "description": "负责数据管理"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439022",
    "name": "数据管理员",
    "code": "data_admin",
    "description": "负责数据管理",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

**错误响应**

| 状态码 | code  | 场景      |
| --- | ----- | ------- |
| 400 | 40004 | 角色代码已存在 |

### 4.2 获取角色列表

**请求**

```
GET /api/roles?page=1&pageSize=10
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "507f1f77bcf86cd799439021",
        "name": "超级管理员",
        "code": "admin",
        "description": "拥有所有权限",
        "permission_count": 12
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
}
```

### 4.3 获取角色详情

**请求**

```
GET /api/roles/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439021",
    "name": "超级管理员",
    "code": "admin",
    "description": "拥有所有权限",
    "permissions": [
      {
        "id": "507f1f77bcf86cd799439031",
        "name": "用户完全控制",
        "code": "user:*"
      }
    ],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 4.4 更新角色

**请求**

```
PUT /api/roles/:id
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "高级管理员",
  "description": "更新后的描述"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439021",
    "name": "高级管理员",
    "code": "admin",
    "description": "更新后的描述",
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

### 4.5 删除角色

**请求**

```
DELETE /api/roles/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

### 4.6 分配权限

**请求**

```
POST /api/roles/:id/permissions
Authorization: Bearer {token}
Content-Type: application/json

{
  "permission_ids": ["507f1f77bcf86cd799439031", "507f1f77bcf86cd799439032"]
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439021",
    "name": "超级管理员",
    "permissions": [...]
  }
}
```

### 4.7 移除权限

**请求**

```
DELETE /api/roles/:id/permissions/:permissionId
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

### 4.8 获取角色权限

**请求**

```
GET /api/roles/:id/permissions
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": "507f1f77bcf86cd799439031",
      "name": "用户完全控制",
      "code": "user:*",
      "module": "user"
    }
  ]
}
```

## 5. 权限管理接口

### 5.1 创建权限

**请求**

```
POST /api/permissions
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "数据查看",
  "code": "data:read",
  "description": "查看业务数据",
  "module": "data"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439031",
    "name": "数据查看",
    "code": "data:read",
    "description": "查看业务数据",
    "module": "data",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

**错误响应**

| 状态码 | code  | 场景      |
| --- | ----- | ------- |
| 400 | 40005 | 权限代码已存在 |

### 5.2 获取权限列表

**请求**

```
GET /api/permissions?page=1&pageSize=10&module=user
Authorization: Bearer {token}
```

**Query参数**

| 参数       | 类型     | 必填 | 说明        |
| -------- | ------ | -- | --------- |
| page     | int    | 否  | 页码，默认1    |
| pageSize | int    | 否  | 每页数量，默认10 |
| module   | string | 否  | 模块筛选      |

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "507f1f77bcf86cd799439031",
        "name": "用户完全控制",
        "code": "user:*",
        "description": "用户的完全控制权限",
        "module": "user"
      }
    ],
    "total": 3,
    "page": 1,
    "pageSize": 10
  }
}
```

### 5.3 获取权限详情

**请求**

```
GET /api/permissions/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439031",
    "name": "用户完全控制",
    "code": "user:*",
    "description": "用户的完全控制权限",
    "module": "user",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 5.4 更新权限

**请求**

```
PUT /api/permissions/:id
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "用户完全控制(更新)",
  "description": "更新后的描述"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439031",
    "name": "用户完全控制(更新)",
    "code": "user:*",
    "description": "更新后的描述",
    "module": "user",
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

### 5.5 删除权限

**请求**

```
DELETE /api/permissions/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

## 6. 业务数据接口

### 6.1 创建业务数据

**请求**

```
POST /api/business
Authorization: Bearer {token}
Content-Type: application/json

{
  "module": "products",
  "data": {
    "name": "产品A",
    "price": 100,
    "stock": 50
  }
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439040",
    "module": "products",
    "data": {
      "name": "产品A",
      "price": 100,
      "stock": 50
    },
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 6.2 按模块查询数据

**请求**

```
GET /api/business/module/:module?page=1&pageSize=10&jql=name contains 'A'
Authorization: Bearer {token}
```

**Query参数**

| 参数       | 类型     | 必填 | 说明        |
| -------- | ------ | -- | --------- |
| page     | int    | 否  | 页码，默认1    |
| pageSize | int    | 否  | 每页数量，默认10 |
| jql      | string | 否  | JQL查询条件   |

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "507f1f77bcf86cd799439040",
        "module": "products",
        "data": {
          "name": "产品A",
          "price": 100,
          "stock": 50
        },
        "created_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
}
```

### 6.3 获取数据详情

**请求**

```
GET /api/business/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439040",
    "module": "products",
    "data": {
      "name": "产品A",
      "price": 100,
      "stock": 50
    },
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 6.4 更新业务数据

**请求**

```
PUT /api/business/:id
Authorization: Bearer {token}
Content-Type: application/json

{
  "data": {
    "name": "产品A(更新)",
    "price": 120,
    "stock": 45
  }
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439040",
    "module": "products",
    "data": {
      "name": "产品A(更新)",
      "price": 120,
      "stock": 45
    },
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

### 6.5 删除业务数据(软删除)

**请求**

```
DELETE /api/business/:id
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

## 7. 集合管理接口

### 7.1 创建集合

**请求**

```
POST /api/collections
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "产品",
  "code": "products",
  "description": "产品数据集合",
  "fields": [
    {"name": "产品名称", "code": "name", "type": "string", "required": true},
    {"name": "价格", "code": "price", "type": "number", "required": true},
    {"name": "库存", "code": "stock", "type": "number", "required": false}
  ]
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "name": "产品",
    "code": "products",
    "description": "产品数据集合",
    "fields": [...],
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 7.2 获取集合列表

**请求**

```
GET /api/collections
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "name": "产品",
      "code": "products",
      "description": "产品数据集合",
      "field_count": 3
    }
  ]
}
```

### 7.3 获取集合详情

**请求**

```
GET /api/collections/:module
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "name": "产品",
    "code": "products",
    "description": "产品数据集合",
    "fields": [
      {"name": "产品名称", "code": "name", "type": "string", "required": true},
      {"name": "价格", "code": "price", "type": "number", "required": true},
      {"name": "库存", "code": "stock", "type": "number", "required": false}
    ]
  }
}
```

### 7.4 创建索引

**请求**

```
POST /api/collections/:module/indexes
Authorization: Bearer {token}
Content-Type: application/json

{
  "fields": ["name"],
  "unique": true,
  "name": "idx_products_name"
}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "name": "idx_products_name",
    "fields": ["name"],
    "unique": true
  }
}
```

### 7.5 获取索引列表

**请求**

```
GET /api/collections/:module/indexes
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "name": "_id_",
      "fields": ["_id"],
      "unique": true
    },
    {
      "name": "idx_products_name",
      "fields": ["name"],
      "unique": true
    }
  ]
}
```

### 7.6 删除索引

**请求**

```
DELETE /api/collections/:module/indexes/:name
Authorization: Bearer {token}
```

**成功响应**

```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

## 8. 错误码定义

### 8.1 通用错误码

| code | 消息                    | 说明     |
| ---- | --------------------- | ------ |
| 200  | success               | 成功     |
| 400  | Bad Request           | 请求参数错误 |
| 401  | Unauthorized          | 未认证    |
| 403  | Forbidden             | 无权限    |
| 404  | Not Found             | 资源不存在  |
| 500  | Internal Server Error | 服务器错误  |

### 8.2 业务错误码

| code  | 消息       | 说明         |
| ----- | -------- | ---------- |
| 40001 | 用户名已存在   | 创建用户时用户名冲突 |
| 40002 | 邮箱已存在    | 创建用户时邮箱冲突  |
| 40003 | 密码格式不正确  | 密码不符合要求    |
| 40004 | 角色代码已存在  | 创建角色时代码冲突  |
| 40005 | 权限代码已存在  | 创建权限时代码冲突  |
| 40006 | 用户不存在    | 操作的用户不存在   |
| 40007 | 角色不存在    | 操作的角色不存在   |
| 40008 | 权限不存在    | 操作的权限不存在   |
| 40009 | 集合不存在    | 操作的集合不存在   |
| 40010 | 数据不存在    | 操作的数据不存在   |
| 40101 | Token无效  | Token解析失败  |
| 40102 | Token已过期 | Token超过有效期 |
| 40301 | 无权限访问    | 权限检查失败     |

## 9. JQL查询语法

### 9.1 支持的操作符

| 操作符          | 说明    | 示例                              |
| ------------ | ----- | ------------------------------- |
| =            | 等于    | name = '产品A'                    |
| !=           | 不等于   | status != 'deleted'             |
| contains     | 包含    | name contains '产品'              |
| not contains | 不包含   | name not contains '测试'          |
| starts with  | 开头匹配  | code starts with 'user\_'       |
| ends with    | 结尾匹配  | code ends with '\_admin'        |
| >            | 大于    | price > 100                     |
| <            | 小于    | price < 100                     |
| >=           | 大于等于  | price >= 100                    |
| <=           | 小于等于  | price <= 100                    |
| in           | 在列表中  | status in ('active', 'pending') |
| not in       | 不在列表中 | status not in ('deleted')       |
| is null      | 为空    | address is null                 |
| is not null  | 不为空   | address is not null             |

### 9.2 逻辑操作符

| 操作符 | 说明  | 示例                                                  |
| --- | --- | --------------------------------------------------- |
| and | 且   | status = 'active' and price > 100                   |
| or  | 或   | name = 'A' or name = 'B'                            |
| ( ) | 优先级 | (status = 'active') and (price > 100 or price < 50) |

### 9.3 示例

```
# 查询名称包含"产品"且价格大于100的数据
name contains '产品' and price > 100

# 查询状态为active或pending的数据
status in ('active', 'pending')

# 查询名称为"产品A"或"产品B"的数据
name = '产品A' or name = '产品B'
```


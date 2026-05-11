# 集合管理模块 - 需求设计文档

## 1. 需求背景

数据中心管理的业务数据按模块（如电影、音乐、书籍）组织，每个模块相当于一个独立的数据集（MongoDB 集合）。需要提供：
- 模块的创建和维护
- 模块级别的独立权限控制
- 模块的索引管理
- 模块的审计追踪

## 2. 功能需求

### FR-COL-01: 创建集合

管理员可以创建新的数据模块（集合），创建时自动建立该模块的权限体系。

**输入**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 模块名称（唯一） |
| description | string | 否 | 模块描述 |
| datatype_owner | string | 是 | 模块管理员用户名 |

**自动行为**:
1. 创建 5 个模块权限（{module}:read/write/delete/admin/field:admin）
2. 创建 3 个集合角色（Owner/Operator/Tourist）
3. 将 Owner 角色自动分配给 datatype_owner

### FR-COL-02: 管理集合

| 操作 | 权限 | 说明 |
|------|------|------|
| 查看集合列表 | collection:read | 分页查询 |
| 查看集合详情 | collection:read | 按 module 查询 |
| 更新集合描述 | collection:write | 修改描述信息 |
| 变更管理员 | collection:write | 自动转移 Owner 角色 |
| 删除集合 | collection:write | 级联删除所有关联数据 |

### FR-COL-03: 索引管理

支持在动态集合上创建、查看和删除 MongoDB 索引。

| 操作 | 说明 |
|------|------|
| 创建索引 | 指定 keys（字段+方向）和 options（名称、唯一性等） |
| 查看索引 | 列出集合所有索引 |
| 删除索引 | 按索引名称删除 |

### FR-COL-04: 集合级角色管理

每个集合有 3 个预定义角色：

| 角色 | 权限 |
|------|------|
| Owner（集合管理员） | 全部权限：读/写/删/管理/字段管理 |
| Operator（数据操作员） | 读/写/删 |
| Tourist（普通用户） | 仅读 |

管理员可以：
- 查看集合角色列表
- 查看角色分配情况
- 为用户分配/移除集合角色

### FR-COL-05: 审计日志

记录集合级别的关键操作，包括：
- 角色分配/移除
- 操作时间、操作人、IP 地址、User-Agent

## 3. 权限设计

### 3.1 两层 RBAC 的权限优先级

```
检查顺序:
  1. system:admin (系统级超级管理员) → 直接通过
  2. 系统级 Role 中的模块权限 (如 movie:read) → 通过
  3. 集合级角色分配 (CollectionRoleAssignment) → 通过
  4. 以上都不满足 → 403
```

### 3.2 集合权限与系统权限的关系

- 集合权限（如 `movie:read`）存储在系统 RBAC 的 `permissions` 表中
- 集合角色（如 `movieOwner`）同时在 `roles` 和 `collection_roles` 两张表记录
- 分配集合角色时，会同步将系统角色赋予用户

## 4. 非功能需求

| 编号 | 需求 | 说明 |
|------|------|------|
| NFR-COL-01 | 级联删除完整性 | 删除集合时清理所有关联数据 |
| NFR-COL-02 | 角色同步一致性 | 系统 role 和 collection_role 保持同步 |
| NFR-COL-03 | 审计可追溯 | 所有角色操作记录审计日志 |

## 5. 业务流程

### 5.1 创建电影数据模块

```
管理员 POST /api/collections
{
  "module": "movie",
  "description": "电影数据模块",
  "datatype_owner": "admin"
}
      │
      ▼
权限检查: collection:write → ✓
      │
      ▼
1. 插入 collections: { module:"movie", collection_name:"movie_data" }
      │
      ▼
2. 创建权限:
   movie:read, movie:write, movie:delete, movie:admin, movie:field:admin
      │
      ▼
3. 创建系统角色 + 集合角色:
   ├── movieOwner (Owner): 全部 5 个权限
   ├── movieOperator (Operator): 3 个权限
   └── movieTourist (Tourist): 1 个权限
      │
      ▼
4. 查询用户 "admin" → 分配 movieOwner 角色
      │
      ▼
5. 返回 201 Created + collection 对象
```

### 5.2 删除集合

```
管理员 DELETE /api/collections/movie
      │
      ▼
1. 删除集合角色和角色分配
2. 删除 5 个权限
3. 删除 movie_data 集合中的业务数据
4. 删除 movie 相关的字段定义
5. 删除 movie 相关的刮削任务
6. 删除 collections 中的元数据
      │
      ▼
返回 200 OK
```

## 6. 接口定义

### 集合 CRUD

```json
// POST /api/collections
Request:
{
  "module": "movie",
  "description": "电影数据模块",
  "datatype_owner": "admin"
}
Response 201:
{
  "_id": "...",
  "module": "movie",
  "description": "电影数据模块",
  "datatype_owner": "admin",
  "collection_name": "movie_data"
}

// GET /api/collections?page=1&pageSize=10
Response 200:
{
  "data": [...],
  "total": 5,
  "page": 1,
  "pageSize": 10
}
```

### 索引管理

```json
// POST /api/collections/movie/indexes
Request:
{
  "keys": { "custom_fields.title": 1 },
  "options": { "name": "idx_title", "unique": false }
}
Response 200:
{ "message": "Index created successfully" }
```

### 角色管理

```json
// GET /api/collections/movie/roles
Response 200:
{
  "data": [
    {
      "_id": "...",
      "collection_module": "movie",
      "name": "movie集合管理员",
      "code": "movieOwner",
      "type": "owner",
      "permission_ids": ["movie:admin", "movie:read", ...]
    }
  ]
}
```

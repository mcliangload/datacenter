# 集合管理模块 - 技术文档

## 1. 模块概述

集合（Collection）是数据中心系统中业务数据的组织单元。每个集合对应一个业务模块（如 movie、music、book），包含该模块的数据存储、字段定义、索引管理和独立的集合级 RBAC 权限控制。

### 模块位置

```
internal/api/
├── handlers.go                              # 集合 CRUD + 索引管理 handler
└── collection_permission_middleware.go      # 集合权限中间件

internal/models/models.go                    # Collection, CollectionRole 等模型

internal/storage/
├── mongodb_storage.go                       # 集合存储层
└── collection_rbac_storage.go               # 集合 RBAC 存储层

pkg/rbac/
└── collection_rbac.go                       # 集合 RBAC 服务
```

---

## 2. 数据模型

### 2.1 Collection

```go
type Collection struct {
    ID             primitive.ObjectID `json:"_id" bson:"_id"`
    Module         string             `json:"module" bson:"module"`
    Description    string             `json:"description" bson:"description"`
    DatatypeOwner  string             `json:"datatype_owner" bson:"datatype_owner"`
    CollectionName string             `json:"collection_name" bson:"collection_name"`
    BaseModel
}
```

**命名规则**: `CollectionName = module + "_data"`（如 `movie_data`）

### 2.2 CollectionRole

```go
type CollectionRole struct {
    ID               primitive.ObjectID `json:"_id" bson:"_id"`
    CollectionModule string             `json:"collection_module"`
    Name             string             `json:"name"`
    Code             string             `json:"code"`
    Type             string             `json:"type"`
    Description      string             `json:"description"`
    PermissionIDs    []string           `json:"permission_ids"` // 权限 code 列表
    BaseModel
}
```

**角色类型常量**:

| 常量 | 值 | 说明 |
|------|-----|------|
| `CollectionRoleTypeOwner` | `"owner"` | 集合管理员 |
| `CollectionRoleTypeOperator` | `"operator"` | 数据操作员 |
| `CollectionRoleTypeTourist` | `"tourist"` | 普通用户 |

### 2.3 CollectionRoleAssignment

```go
type CollectionRoleAssignment struct {
    ID               primitive.ObjectID `json:"_id" bson:"_id"`
    UserID           string             `json:"user_id"`
    CollectionModule string             `json:"collection_module"`
    CollectionRoleID string             `json:"collection_role_id"`
    BaseModel
}
```

### 2.4 AuditLog

```go
type AuditLog struct {
    ID         primitive.ObjectID `json:"_id" bson:"_id"`
    Timestamp  time.Time          `json:"timestamp"`
    UserID     string             `json:"user_id"`
    Username   string             `json:"username"`
    Action     string             `json:"action"`
    Resource   string             `json:"resource"`
    ResourceID string             `json:"resource_id"`
    Details    string             `json:"details"`
    IPAddress  string             `json:"ip_address"`
    UserAgent  string             `json:"user_agent"`
}
```

---

## 3. 集合级 RBAC 权限体系

### 3.1 权限代码

每个集合创建时自动生成 5 个模块级权限：

| 权限代码 | 含义 |
|----------|------|
| `{module}:read` | 读取模块数据 |
| `{module}:write` | 写入模块数据 |
| `{module}:delete` | 删除模块数据 |
| `{module}:admin` | 管理模块（含角色分配） |
| `{module}:field:admin` | 管理模块字段定义 |

### 3.2 角色权限矩阵

| 能力 | Owner | Operator | Tourist |
|------|-------|----------|---------|
| 读取数据 | :read | :read | :read |
| 写入数据 | :write | :write | - |
| 删除数据 | :delete | :delete | - |
| 管理模块 | :admin | - | - |
| 管理字段 | :field:admin | - | - |

### 3.3 集合权限检查

```go
func (s *CollectionRBACService) CheckCollectionPermission(
    ctx context.Context, userID, module string,
    requiredPermission CollectionPermission,
) (bool, error)
```

**检查流程**:
1. 检查用户系统级角色是否拥有 `system:admin` → 直接通过
2. 检查用户系统级 Role 是否已包含模块权限代码（如 `movie:read`）
3. 查询 `collection_role_assignments` 表，检查用户在指定模块是否有集合角色分配
4. 如果有，检查该集合角色的 `PermissionIDs` 是否包含所需权限

---

## 4. 集合权限中间件

### 4.1 从 URL 参数获取模块名

```go
CollectionPermissionMiddleware(svc, requiredPerm) gin.HandlerFunc
```

用于路径包含 `:module` 参数的路由：

```go
business.GET("/module/:module",
    CollectionPermissionMiddleware(h.collectionRBACService, rbac.CollectionPermissionRead),
    h.GetBusinessDataByModule)
```

### 4.2 从请求体获取模块名

```go
CollectionPermissionMiddlewareFromBody(svc, requiredPerm) gin.HandlerFunc
```

用于 POST/PUT 请求，模块名从 JSON body 的 `module` 字段获取。

### 4.3 先查询字段再检查

```go
CollectionPermissionFieldAdminMiddleware(svc, storage) gin.HandlerFunc
```

用于字段定义的更新和删除，先通过 `storage.GetFieldDefinitionByID(id)` 获取字段所属模块，再检查集合的 `:field:admin` 权限。

---

## 5. API 接口

### 5.1 集合 CRUD

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/collections | collection:read | 集合列表（分页） |
| GET | /api/collections/:module | collection:read | 集合详情 |
| POST | /api/collections | collection:write | 创建集合（自动创建角色和权限） |
| PUT | /api/collections/:module | collection:write | 更新集合（自动转移 Owner） |
| DELETE | /api/collections/:module | collection:write | 级联删除 |

### 5.2 索引管理

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/collections/:module/indexes | 创建索引 |
| GET | /api/collections/:module/indexes | 获取索引列表 |
| DELETE | /api/collections/:module/indexes/:name | 删除索引 |

### 5.3 集合角色管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/collections/:module/roles | 获取集合角色列表 |
| GET | /api/collections/:module/roles/assignments | 获取角色分配列表 |
| POST | /api/collections/:module/roles/assign | 分配集合角色 |
| DELETE | /api/collections/:module/roles/:rid/assignments/:uid | 移除集合角色 |
| GET | /api/collections/:module/audit-logs | 获取审计日志 |

---

## 6. 存储层接口

### 6.1 Collection 存储

```go
CreateCollection(collection *models.Collection) error
GetCollectionByModule(module string) (*models.Collection, error)
GetCollections(skip, limit int64) ([]models.Collection, error)
GetCollectionsCount() (int64, error)
UpdateCollection(collection *models.Collection) error
DeleteCollection(module string) error  // 级联删除
```

### 6.2 索引存储

```go
CreateIndex(collectionName string, keys bson.M, opts *options.IndexOptions) error
GetDynamicCollection(collectionName string) *mongo.Collection
```

### 6.3 集合 RBAC 存储

```go
CreateCollectionRole(role *models.CollectionRole) error
GetCollectionRoleByID(id string) (*models.CollectionRole, error)
GetCollectionRolesByModule(module string) ([]models.CollectionRole, error)
DeleteCollectionRole(id string) error
AssignCollectionRole(assignment *models.CollectionRoleAssignment) error
RemoveCollectionRoleAssignment(userID, module, roleID string) error
GetUserCollectionRole(userID, module string) (*models.CollectionRoleAssignment, error)
GetCollectionRoleAssignments(module string) ([]models.CollectionRoleAssignment, error)
CreateAuditLog(log *models.AuditLog) error
GetAuditLogsByResource(resource, resourceID string, skip, limit int64) ([]models.AuditLog, error)
```

---

## 7. 创建集合流程

```
POST /api/collections { module, description, datatype_owner }
      │
      ▼
1. 在 datacenter.collections 插入集合元数据
      │
      ▼
2. 在 rbac.permissions 创建 5 个模块权限
   ┌─────────────────────────────────────────────┐
   │ {module}:read, {module}:write,              │
   │ {module}:delete, {module}:admin,            │
   │ {module}:field:admin                        │
   └─────────────────────────────────────────────┘
      │
      ▼
3. 在 rbac.roles 创建 3 个系统角色
   ┌─────────────────────────────────────────────┐
   │ {module}Owner    → admin+read+write+delete+field:admin │
   │ {module}Operator → read+write+delete        │
   │ {module}Tourist  → read                     │
   └─────────────────────────────────────────────┘
      │
      ▼
4. 在 rbac.collection_roles 创建 3 个集合角色（权限代码列表）
      │
      ▼
5. 自动将 Owner 角色分配给 datatype_owner 用户
      │
      ▼
返回 201 Created
```

---

## 8. 删除集合流程（级联）

```
DELETE /api/collections/:module
      │
      ▼
1. 删除集合角色和角色分配 (DeleteCollectionRoles)
      │
      ▼
2. 删除模块权限 (5 条 permission)
      │
      ▼
3. 级联删除集合数据 (DeleteCollection):
   ├── 删除字段定义 (field_definitions)
   ├── 删除业务数据 ({module}_data 集合)
   ├── 删除刮削任务 (scrape_tasks)
   └── 删除集合元数据 (collections)
      │
      ▼
返回 200 OK
```

---

## 9. Owner 转移机制

当更新集合的 `datatype_owner` 时：

```
PUT /api/collections/:module { datatype_owner: "newowner" }
      │
      ▼
1. 更新集合文档的 datatype_owner
      │
      ▼
2. IF datatype_owner 变更:
   ├── 查找该集合的 Owner 角色
   ├── 移除旧 owner 的 Owner 角色分配
   └── 赋予新 owner Owner 角色（通过 username 查找用户）
      │
      ▼
返回 200 OK
```

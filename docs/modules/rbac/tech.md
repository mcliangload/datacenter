# RBAC 权限模块 - 技术文档

## 1. 模块概述

RBAC (Role-Based Access Control) 模块实现系统级权限管理，采用经典的 User-Role-Permission 三层多对多模型。支持通配符权限匹配和超级管理员机制。

### 模块位置

```
pkg/rbac/
├── rbac.go              # 系统级 RBAC 服务
└── collection_rbac.go   # 集合级 RBAC 服务

internal/storage/
├── rbac_storage.go      # RBAC 数据访问层 (users/permissions/roles)
└── collection_rbac_storage.go  # 集合 RBAC 存储层

internal/models/models.go  # 数据模型 (User/Permission/Role/CollectionRole 等)
```

---

## 2. 数据模型

### 2.1 ER 关系

```
User (users)
  ├── _id, username, password, email
  └── role_ids: [string]          ──┐ N:M
                                    │
Role (roles)                        │
  ├── _id, name, code, description  │
  └── permission_ids: [string]     ──┤ N:M
                                    │
Permission (permissions)            │
  ├── _id, name, code, description◀─┘
```

### 2.2 User

```go
type User struct {
    ID       primitive.ObjectID `json:"_id" bson:"_id"`
    Username string             `json:"username" bson:"username"`
    Password string             `json:"password,omitempty" bson:"password"`
    Email    string             `json:"email" bson:"email"`
    RoleIDs  []string           `json:"role_ids" bson:"role_ids"`
    BaseModel
}
```

### 2.3 Permission

```go
type Permission struct {
    ID          primitive.ObjectID `json:"_id" bson:"_id"`
    Name        string             `json:"name" bson:"name"`
    Code        string             `json:"code" bson:"code"`
    Description string             `json:"description" bson:"description"`
    BaseModel
}
```

### 2.4 Role

```go
type Role struct {
    ID            primitive.ObjectID `json:"_id" bson:"_id"`
    Name          string             `json:"name" bson:"name"`
    Code          string             `json:"code" bson:"code"`
    Description   string             `json:"description" bson:"description"`
    PermissionIDs []string           `json:"permission_ids" bson:"permission_ids"`
    BaseModel
}
```

---

## 3. Service 接口

### 3.1 Service 结构

```go
type Service struct {
    storage storage.RBACStorage
}

func NewService(s storage.RBACStorage) *Service
```

### 3.2 核心方法

| 方法 | 签名 | 说明 |
|------|------|------|
| `CheckPermission` | `(ctx, userID, requiredPermission Permission) (bool, error)` | 检查用户是否拥有某权限 |
| `GetUserPermissions` | `(ctx, userID) ([]string, error)` | 获取用户全部权限 code |
| `HasAnyPermission` | `(ctx, userID, permissions []Permission) (bool, error)` | 是否拥有任一权限 |
| `HasAllPermissions` | `(ctx, userID, permissions []Permission) (bool, error)` | 是否拥有全部权限 |
| `GetAPIPermission` | `(method, path string) Permission` | 根据 HTTP 方法+路径推断所需权限 |

### 3.3 权限常量

```go
type Permission string

const (
    PermissionUserRead       Permission = "user:read"
    PermissionUserWrite      Permission = "user:write"
    PermissionRoleRead       Permission = "role:read"
    PermissionRoleWrite      Permission = "role:write"
    PermissionPermissionRead Permission = "permission:read"
    PermissionPermissionWrite Permission = "permission:write"
    PermissionDataRead       Permission = "data:read"
    PermissionDataWrite      Permission = "data:write"
    PermissionFieldRead      Permission = "field:read"
    PermissionFieldWrite     Permission = "field:write"
    PermissionScrapeRead     Permission = "scrape:read"
    PermissionScrapeWrite    Permission = "scrape:write"
    PermissionCollectionRead Permission = "collection:read"
    PermissionCollectionWrite Permission = "collection:write"
    PermissionSystemAdmin    Permission = "system:admin"
    // ... 还有对应的 :manage 变体
)
```

---

## 4. 权限检查算法

### 4.1 CheckPermission 流程

```
CheckPermission(userID, requiredPermission):
  1. 查询用户 → user
  2. 遍历 user.RoleIDs
     2a. 查询 Role → role
     2b. 遍历 role.PermissionIDs
         2b-i. 查询 Permission → perm
         2b-ii. IF perm.Code == "system:admin" → return true (超级管理员)
         2b-iii. IF matchPermission(perm.Code, requiredPermission) → return true
  3. return false
```

时间复杂度: O(R × P)，其中 R=用户角色数，P=角色平均权限数

### 4.2 matchPermission 通配符匹配

```go
func (s *Service) matchPermission(userPermCode, requiredPermCode string) bool {
    // 精确匹配
    if userPermCode == requiredPermCode {
        return true
    }
    // 通配符匹配: "user:*" 匹配 "user:read", "user:write" 等
    if strings.HasSuffix(userPermCode, ":*") {
        prefix := strings.TrimSuffix(userPermCode, "*")
        if strings.HasPrefix(requiredPermCode, prefix) {
            return true
        }
    }
    return false
}
```

### 4.3 超级管理员

`system:admin` 是一个特殊权限码，拥有此权限的用户在 `CheckPermission` 中直接返回 `true`，绕过所有其他权限检查。

---

## 5. 存储层 (RBACStorage)

### 5.1 接口定义

```go
type RBACStorage interface {
    // User CRUD
    CreateUser(user *models.User) error
    GetUserByID(id string) (*models.User, error)
    GetUserByUsername(username string) (*models.User, error)
    GetUsers(skip, limit int64) ([]models.User, error)
    GetUsersCount() (int64, error)
    UpdateUser(user *models.User) error
    DeleteUser(id string) error
    AssignRoleToUser(userID, roleID, operatorID string) error
    RemoveRoleFromUser(userID, roleID string) error

    // Permission CRUD
    CreatePermission(permission *models.Permission) error
    GetPermissionByID(id string) (*models.Permission, error)
    GetPermissionByCode(code string) (*models.Permission, error)
    GetPermissions(skip, limit int64) ([]models.Permission, error)
    GetPermissionsCount() (int64, error)
    UpdatePermission(permission *models.Permission) error
    DeletePermission(id string) error

    // Role CRUD
    CreateRole(role *models.Role) error
    GetRoleByID(id string) (*models.Role, error)
    GetRoleByCode(code string) (*models.Role, error)
    GetRoles(skip, limit int64) ([]models.Role, error)
    GetRolesCount() (int64, error)
    UpdateRole(role *models.Role) error
    DeleteRole(id string) error
}
```

### 5.2 索引

| 集合 | 索引 | 类型 |
|------|------|------|
| users | `{username: 1}` | 唯一索引 |
| users | `{email: 1}` | 唯一索引 |
| permissions | `{code: 1}` | 唯一索引 |
| roles | `{code: 1}` | 唯一索引 |

---

## 6. 权限中间件

### 6.1 PermissionMiddleware

```go
func (h *Handler) PermissionMiddleware(requiredPerm rbac.Permission) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(403, gin.H{"error": "No user ID found"})
            c.Abort()
            return
        }
        hasPermission, err := h.rbacService.CheckPermission(
            c.Request.Context(), userID.(string), requiredPerm)
        if err != nil || !hasPermission {
            c.JSON(403, gin.H{"error": "Permission denied: " + string(requiredPerm)})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 6.2 中间件使用方式

路由注册时使用 Gin 的 `Use()` 方法在路由级别附加权限检查：

```go
// 所有用户路由需要 user:read 权限
users := protected.Group("/users")
users.Use(h.PermissionMiddleware(rbac.PermissionUserRead))
{
    users.GET("", h.GetUsers)      // GET 继承 user:read
    users.POST("", h.CreateUser)   // POST 需要额外 user:write
        .Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
}
```

# RBAC 权限模块 - 需求实现文档

## 1. 实现概述

RBAC 模块分为两层：
- **服务层** (`pkg/rbac/rbac.go`): 权限检查逻辑
- **存储层** (`internal/storage/rbac_storage.go`): MongoDB 数据访问

---

## 2. 文件清单

| 文件 | 说明 |
|------|------|
| `pkg/rbac/rbac.go` | RBAC 服务：权限检查、通配符匹配、权限查询 |
| `internal/storage/rbac_storage.go` | RBAC 存储层：User/Permission/Role CRUD |
| `internal/models/models.go` | 数据模型：User、Permission、Role |
| `internal/api/handlers.go` | Handler：路由注册、PermissionMiddleware、CRUD handler |
| `cmd/server/main.go` | RBAC 服务初始化 |

---

## 3. 服务初始化

```go
// cmd/server/main.go:79-80
rbacService := rbac.NewService(rbacStorage)
```

---

## 4. 权限检查核心实现

### 4.1 CheckPermission

```go
// pkg/rbac/rbac.go:63-99
func (s *Service) CheckPermission(ctx context.Context, userID string, requiredPermission Permission) (bool, error) {
    user, err := s.storage.GetUserByID(userID)
    if err != nil {
        return false, err
    }

    if len(user.RoleIDs) == 0 {
        return false, nil
    }

    for _, roleID := range user.RoleIDs {
        role, err := s.storage.GetRoleByID(roleID)
        if err != nil {
            continue
        }

        for _, permID := range role.PermissionIDs {
            perm, err := s.storage.GetPermissionByID(permID)
            if err != nil {
                continue
            }

            // 超级管理员：直接通过
            if perm.Code == string(PermissionSystemAdmin) {
                return true, nil
            }

            // 权限匹配
            if s.matchPermission(perm.Code, string(requiredPermission)) {
                return true, nil
            }
        }
    }

    return false, nil
}
```

### 4.2 通配符匹配

```go
// pkg/rbac/rbac.go:101-114
func (s *Service) matchPermission(userPermCode, requiredPermCode string) bool {
    // 精确匹配
    if userPermCode == requiredPermCode {
        return true
    }

    // 通配符匹配: "user:*" 匹配 "user:read"
    if strings.HasSuffix(userPermCode, ":*") {
        prefix := strings.TrimSuffix(userPermCode, "*")
        if strings.HasPrefix(requiredPermCode, prefix) {
            return true
        }
    }

    return false
}
```

### 4.3 GetUserPermissions

```go
// pkg/rbac/rbac.go:116-145
func (s *Service) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
    user, err := s.storage.GetUserByID(userID)
    if err != nil {
        return nil, err
    }

    permMap := make(map[string]bool)
    for _, roleID := range user.RoleIDs {
        role, err := s.storage.GetRoleByID(roleID)
        if err != nil {
            continue
        }
        for _, permID := range role.PermissionIDs {
            perm, err := s.storage.GetPermissionByID(permID)
            if err != nil {
                continue
            }
            permMap[perm.Code] = true
        }
    }

    perms := make([]string, 0, len(permMap))
    for p := range permMap {
        perms = append(perms, p)
    }
    return perms, nil
}
```

---

## 5. 中间件实现

### 5.1 PermissionMiddleware

```go
// internal/api/handlers.go:295-314
func (h *Handler) PermissionMiddleware(requiredPerm rbac.Permission) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{"error": "No user ID found"})
            c.Abort()
            return
        }

        hasPermission, err := h.rbacService.CheckPermission(
            c.Request.Context(), userID.(string), requiredPerm)
        if err != nil || !hasPermission {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Permission denied: " + string(requiredPerm),
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## 6. API → 权限映射

### 6.1 静态映射函数

```go
// pkg/rbac/rbac.go:173-249
func GetAPIPermission(method, path string) Permission {
    // 根据 HTTP 方法和路径推断所需权限
    if strings.HasPrefix(path, "users") {
        switch method {
        case "GET":  return PermissionUserRead
        case "POST", "PUT", "DELETE": return PermissionUserWrite
        }
    }
    if strings.HasPrefix(path, "roles") {
        switch method {
        case "GET":  return PermissionRoleRead
        case "POST", "PUT", "DELETE": return PermissionRoleWrite
        }
    }
    // ... 其他资源类似
}
```

### 6.2 路由注册中的权限绑定

```go
// internal/api/handlers.go:49-63
protected := r.Group("/api")
protected.Use(h.AuthMiddleware())
{
    users := protected.Group("/users")
    users.Use(h.PermissionMiddleware(rbac.PermissionUserRead))  // 组级：全部需要 user:read
    {
        users.GET("", h.GetUsers)                              // 继承 user:read
        users.POST("", h.CreateUser)
            .Use(h.PermissionMiddleware(rbac.PermissionUserWrite)) // 额外：需要 user:write
        users.PUT("/:id", h.UpdateUser)
            .Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
        users.DELETE("/:id", h.DeleteUser)
            .Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
        // ...
    }
}
```

---

## 7. User CRUD 实现

### 7.1 获取用户列表（含密码清除）

```go
// internal/api/handlers.go:316-358
func (h *Handler) GetUsers(c *gin.Context) {
    // 支持 skip/limit 和 page/pageSize 两种分页参数
    skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
    limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

    // page/pageSize 优先
    if pageStr := c.Query("page"); pageStr != "" {
        // 计算 skip = (page-1)*pageSize
    }

    users, _ := h.rbacStorage.GetUsers(skip, limit)
    total, _ := h.rbacStorage.GetUsersCount()

    // 安全措施：清除所有 password 字段
    for i := range users {
        users[i].Password = ""
    }

    c.JSON(200, gin.H{
        "data": users, "total": total,
        "page": page, "pageSize": limit,
    })
}
```

### 7.2 创建用户

```go
// internal/api/handlers.go:371-404
func (h *Handler) CreateUser(c *gin.Context) {
    var req struct {
        Username string   `json:"username" binding:"required"`
        Password string   `json:"password" binding:"required"`
        Email    string   `json:"email" binding:"required"`
        RoleIDs  []string `json:"role_ids"`
    }
    // 1. 解析请求
    // 2. bcrypt 哈希密码
    // 3. 构造 User{Username, Password:hash, Email, RoleIDs}
    // 4. rbacStorage.CreateUser(user)
    // 5. 清除密码后返回 201
}
```

### 7.3 分配/移除角色

```go
// AssignRoleToUser
func (h *Handler) AssignRoleToUser(c *gin.Context) {
    userID := c.Param("id")
    var req struct { RoleID string `json:"role_id"` }

    user, _ := h.rbacStorage.GetUserByID(userID)
    // 去重检查
    for _, roleID := range user.RoleIDs {
        if roleID == req.RoleID {
            return 400 "Role already assigned"
        }
    }
    user.RoleIDs = append(user.RoleIDs, req.RoleID)
    h.rbacStorage.UpdateUser(user)
}

// RemoveRoleFromUser
func (h *Handler) RemoveRoleFromUser(c *gin.Context) {
    userID, roleID := c.Param("id"), c.Param("roleId")
    user, _ := h.rbacStorage.GetUserByID(userID)
    // 过滤掉目标 roleID
    newRoleIDs := []string{}
    for _, id := range user.RoleIDs {
        if id != roleID {
            newRoleIDs = append(newRoleIDs, id)
        }
    }
    user.RoleIDs = newRoleIDs
    h.rbacStorage.UpdateUser(user)
}
```

---

## 8. 存储层实现

### 8.1 MongoDB 连接

RBAC 存储使用独立的 MongoDB 数据库 `rbac`，与业务数据库 `datacenter` 物理隔离。

```go
// cmd/server/main.go:51-54
rbacStorage, err := storage.NewRBACMongoDBStorage(
    getEnv("MONGODB_RBAC_URI", "mongodb://localhost:27017"),
    getEnv("MONGODB_RBAC_DATABASE", "rbac"),
)
```

### 8.2 默认数据初始化

```go
// cmd/server/main.go:60-62
if err := rbacStorage.InitDefaultData(); err != nil {
    panic("Failed to initialize default data: " + err.Error())
}
```

`InitDefaultData()` 在 `internal/storage/rbac_storage.go` 中实现，创建：
- 系统默认权限（22 个）
- 默认角色（admin + user）
- 默认管理员用户（admin, bcrypt 哈希密码）

### 8.3 数据库索引

在 `NewRBACMongoDBStorage` 中创建：

```go
// users 集合
{username: 1} → 唯一索引
{email: 1}    → 唯一索引

// permissions 集合
{code: 1} → 唯一索引

// roles 集合
{code: 1} → 唯一索引
```

---

## 9. 安全措施

| 措施 | 实现位置 | 说明 |
|------|----------|------|
| 密码清除 | `GetUsers()`, `GetUserByID()`, `CreateUser()`, `UpdateUser()` | 所有返回 User 的响应中 `user.Password = ""` |
| 密码哈希 | `CreateUser()`, `UpdateUser()`, `Register()` | `auth.HashPassword()` bcrypt |
| 超级管理员 | `CheckPermission()` | `system:admin` 直接返回 true |
| 403 统一错误 | `PermissionMiddleware()` | 权限不足统一返回 403 |

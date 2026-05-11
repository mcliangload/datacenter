# 集合管理模块 - 需求实现文档

## 1. 实现概述

集合管理模块涉及 3 个层面的实现：
- **Handler** (`internal/api/handlers.go`): HTTP 请求处理
- **Service** (`pkg/rbac/collection_rbac.go`): 集合权限逻辑
- **Storage** (`internal/storage/`): 数据库操作

---

## 2. 文件清单

| 文件 | 说明 |
|------|------|
| `internal/api/handlers.go` | 集合/索引/角色管理 handler |
| `internal/api/collection_permission_middleware.go` | 集合权限中间件 |
| `pkg/rbac/collection_rbac.go` | 集合 RBAC 服务 |
| `internal/storage/mongodb_storage.go` | 集合 CRUD 存储 |
| `internal/storage/collection_rbac_storage.go` | 集合 RBAC 存储 |
| `internal/models/models.go` | Collection、CollectionRole 等模型 |
| `cmd/server/main.go` | 集合 RBAC 服务初始化 |

---

## 3. 创建集合实现

### 3.1 Handler

```go
// internal/api/handlers.go:1422-1478
func (h *Handler) CreateCollection(c *gin.Context) {
    var req struct {
        Module        string `json:"module" binding:"required"`
        Description   string `json:"description"`
        DatatypeOwner string `json:"datatype_owner" binding:"required"`
    }
    // 1. 构造 Collection
    collection := &models.Collection{
        Module:         req.Module,
        Description:    req.Description,
        DatatypeOwner:  req.DatatypeOwner,
        CollectionName: req.Module + "_data",           // 命名规则
        CreatedBy:      userIDStr,
    }

    // 2. 保存集合元数据
    if err := h.storage.CreateCollection(collection); err != nil {
        return 500
    }

    // 3. 创建集合角色和权限（失败时回滚集合）
    if err := h.collectionRBACService.CreateCollectionRoles(
        ctx, req.Module, collection.CreatedBy); err != nil {
        h.storage.DeleteCollection(req.Module) // 回滚
        return 500
    }

    // 4. 自动分配 Owner 角色给 datatype_owner
    roles, _ := h.collectionRBACService.GetCollectionRoles(ctx, req.Module)
    for _, role := range roles {
        if role.Type == models.CollectionRoleTypeOwner {
            ownerUser, _ := h.rbacStorage.GetUserByUsername(req.DatatypeOwner)
            if ownerUser != nil {
                h.collectionRBACService.AssignCollectionRole(
                    ctx, ownerUser.ID.Hex(), req.Module, role.ID.Hex(), collection.CreatedBy)
            }
        }
    }
    c.JSON(http.StatusCreated, collection)
}
```

**关键设计**: 
- 集合名称固定为 `{module}_data`
- 如果创建角色失败，回滚已创建的集合
- Owner 自动分配优先通过 username 查找用户

### 3.2 CreateCollectionRoles

```go
// pkg/rbac/collection_rbac.go:92-193
func (s *CollectionRBACService) CreateCollectionRoles(ctx context.Context, module, operatorID string) error {
    // 1. 创建 5 个权限
    permDefs := []struct{ Code, Name string }{
        {module + ":read", module + " 读取"},
        {module + ":write", module + " 写入"},
        {module + ":delete", module + " 删除"},
        {module + ":admin", module + " 管理"},
        {module + ":field:admin", module + " 字段管理"},
    }
    for _, pd := range permDefs {
        perm := &models.Permission{Name: pd.Name, Code: pd.Code, ...}
        s.rbacStorage.CreatePermission(perm)
        createdPermIDs[pd.Code] = perm.ID.Hex()
    }

    // 2. 创建 3 个系统角色 + 3 个集合角色
    roleTemplates := []roleTemplate{
        {Type: "owner", Code: module + "Owner", PermCodes: [
            module+":admin", module+":read", module+":write",
            module+":delete", module+":field:admin"]},
        {Type: "operator", Code: module + "Operator", PermCodes: [
            module+":read", module+":write", module+":delete"]},
        {Type: "tourist", Code: module + "Tourist", PermCodes: [
            module+":read"]},
    }
    for _, rt := range roleTemplates {
        // 创建系统角色 (rbac.roles)
        sysRole := &models.Role{Name: rt.Name, Code: rt.Code, PermissionIDs: ...}
        s.rbacStorage.CreateRole(sysRole)

        // 创建集合角色 (rbac.collection_roles)
        colRole := &models.CollectionRole{CollectionModule: module, Permissions: permCodes, ...}
        s.collectionRBACStorage.CreateCollectionRole(colRole)
    }
    return nil
}
```

---

## 4. 删除集合实现

```go
// internal/api/handlers.go:1544-1576
func (h *Handler) DeleteCollection(c *gin.Context) {
    module := c.Param("module")
    ctx := c.Request.Context()

    // 1. 删除集合角色和角色分配
    if err := h.collectionRBACService.DeleteCollectionRoles(ctx, module); err != nil {
        return 500
    }

    // 2. 删除模块权限（5 个）
    permCodes := []string{
        module + ":read", module + ":write", module + ":delete",
        module + ":admin", module + ":field:admin",
    }
    for _, code := range permCodes {
        perm, err := h.rbacStorage.GetPermissionByCode(code)
        if err == nil && perm != nil {
            h.rbacStorage.DeletePermission(perm.ID.Hex())
        }
    }

    // 3. 级联删除集合数据
    if err := h.storage.DeleteCollection(module); err != nil {
        return 500
    }

    c.JSON(http.StatusOK, gin.H{"message": "Collection deleted successfully"})
}
```

---

## 5. 更新集合（含 Owner 转移）

```go
// internal/api/handlers.go:1480-1542
func (h *Handler) UpdateCollection(c *gin.Context) {
    module := c.Param("module")
    // ... 解析请求 ...

    oldOwner := collection.DatatypeOwner
    collection.Description = req.Description       // 可选更新
    collection.DatatypeOwner = req.DatatypeOwner   // 可选更新
    h.storage.UpdateCollection(collection)

    // Owner 变更时自动转移角色
    if req.DatatypeOwner != "" && req.DatatypeOwner != oldOwner {
        roles, _ := h.collectionRBACService.GetCollectionRoles(ctx, module)
        for _, role := range roles {
            if role.Type == models.CollectionRoleTypeOwner {
                // 移除旧 Owner
                if oldOwner != "" {
                    oldUser, _ := h.rbacStorage.GetUserByUsername(oldOwner)
                    if oldUser != nil {
                        h.collectionRBACService.RemoveCollectionRole(
                            ctx, oldUser.ID.Hex(), module, role.ID.Hex(), "system")
                    }
                }
                // 赋予新 Owner
                newUser, _ := h.rbacStorage.GetUserByUsername(req.DatatypeOwner)
                if newUser != nil {
                    h.collectionRBACService.AssignCollectionRole(
                        ctx, newUser.ID.Hex(), module, role.ID.Hex(), operatorID)
                }
            }
        }
    }
}
```

---

## 6. 集合权限中间件实现

### 6.1 从 URL 参数获取模块

```go
// internal/api/collection_permission_middleware.go
func CollectionPermissionMiddleware(
    svc *rbac.CollectionRBACService,
    requiredPerm rbac.CollectionPermission,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        module := c.Param("module")  // 从 URL 路径提取
        userID, _ := c.Get("user_id")

        hasPermission, _ := svc.CheckCollectionPermission(
            c.Request.Context(), userID.(string), module, requiredPerm)

        if !hasPermission {
            c.JSON(403, gin.H{"error": "Permission denied"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 6.2 从请求体获取模块

```go
func CollectionPermissionMiddlewareFromBody(
    svc *rbac.CollectionRBACService,
    requiredPerm rbac.CollectionPermission,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 先读取 body 获取 module 字段
        var body struct{ Module string `json:"module"` }
        // ... 解析 body 并恢复 ...
        module := body.Module
        // ... 权限检查 ...
    }
}
```

---

## 7. 分配/移除集合角色实现

### 7.1 AssignCollectionRole

```go
// pkg/rbac/collection_rbac.go:217-238
func (s *CollectionRBACService) AssignCollectionRole(
    ctx context.Context, userID, module, roleID, operatorID string,
) error {
    // 1. 创建集合角色分配记录
    assignment := &models.CollectionRoleAssignment{
        UserID:           userID,
        CollectionModule: module,
        CollectionRoleID: roleID,
        CreatedBy:        operatorID,
    }
    s.collectionRBACStorage.AssignCollectionRole(assignment)

    // 2. 同步赋予系统角色（rbac.roles → user.role_ids）
    colRole, _ := s.collectionRBACStorage.GetCollectionRoleByID(roleID)
    if colRole != nil {
        sysRole, _ := s.rbacStorage.GetRoleByCode(colRole.Code)
        if sysRole != nil {
            s.rbacStorage.AssignRoleToUser(userID, sysRole.ID.Hex(), operatorID)
        }
    }
    return nil
}
```

### 7.2 RemoveCollectionRole

```go
func (s *CollectionRBACService) RemoveCollectionRole(
    ctx context.Context, userID, module, roleID, operatorID string,
) error {
    // 1. 先移除系统角色
    colRole, _ := s.collectionRBACStorage.GetCollectionRoleByID(roleID)
    if colRole != nil {
        sysRole, _ := s.rbacStorage.GetRoleByCode(colRole.Code)
        if sysRole != nil {
            s.rbacStorage.RemoveRoleFromUser(userID, sysRole.ID.Hex())
        }
    }
    // 2. 删除集合角色分配
    return s.collectionRBACStorage.RemoveCollectionRoleAssignment(userID, module, roleID)
}
```

---

## 8. 审计日志

```go
// pkg/rbac/collection_rbac.go:280-293
func (s *CollectionRBACService) LogAction(
    ctx context.Context,
    userID, username, action, resource, resourceID, details,
    ipAddress, userAgent string,
) error {
    log := &models.AuditLog{
        UserID: userID, Username: username,
        Action: action, Resource: resource,
        ResourceID: resourceID, Details: details,
        IPAddress: ipAddress, UserAgent: userAgent,
    }
    return s.collectionRBACStorage.CreateAuditLog(log)
}
```

审计日志在角色分配/移除时自动记录，包含操作人、目标用户、操作类型、IP 等完整信息。

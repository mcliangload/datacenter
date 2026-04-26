# 任务列表

## 任务 1: 恢复并重构 collection_rbac_storage.go 文件
- [x] 创建新的 `internal/storage/collection_rbac_storage.go` 文件，包含 `CollectionRBACStorage` 接口和 `collectionRBACStorage` 实现
  - [x] 接口方法：CreateCollectionRole, GetCollectionRoleByID, GetCollectionRolesByModule, UpdateCollectionRole, DeleteCollectionRole, GetCollectionRoleByType
  - [x] 接口方法：AssignCollectionRole, RemoveCollectionRoleAssignment, GetCollectionRoleAssignments, GetUserCollectionRoles, GetUserCollectionRole
  - [x] 接口方法：CreateAuditLog, GetAuditLogs, GetAuditLogsByUser, GetAuditLogsByResource
- [x] 验证编译通过

## 任务 2: 重构角色类型常量 (models.go)
- [x] `CollectionRoleTypeOwner = "owner"` 替换 `CollectionRoleTypeAdmin = "admin"`
- [x] `CollectionRoleTypeOperator = "operator"` 替换 `CollectionRoleTypeDataAdmin = "data_admin"` 和 `CollectionRoleTypeOperator`
- [x] `CollectionRoleTypeTourist = "tourist"` 替换 `CollectionRoleTypeUser = "user"`
- [x] 删除旧的 `CollectionRoleTypeDataAdmin`

## 任务 3: 重写 CreateCollectionRoles - 创建正式权限文档和角色
- [x] 在 `CreateCollectionRoles` 中，先在 `rbac.permissions` 集合创建5个权限文档：
  - `{module}:read`, `{module}:write`, `{module}:delete`, `{module}:admin`, `{module}:field:admin`
- [x] 创建3个角色，使用新的命名规范并引用权限code：
  - **Owner** (`{module}Owner`): 包含 admin + read + write + delete + field:admin
  - **Operator** (`{module}Operator`): 包含 read + write + delete
  - **Tourist** (`{module}Tourist`): 包含 read
- [x] 更新角色名称格式：`%sOwner`, `%sOperator`, `%sTourist`
- [x] 更新CheckCollectionPermission方法，支持通过正式权限code匹配

## 任务 4: 更新 InitDefaultData 增强MongoDB内置测试数据
- [x] 在 `InitDefaultData()` 中检查是否已有数据，如无则创建：
  - [x] 创建 `system:admin` 权限
  - [x] 创建超级管理员角色 `root`，包含所有21个系统权限
  - [x] 创建管理员用户 `admin`（密码：`liangminchuan`），关联 root 角色

## 任务 5: 适配 handlers.go 中的引用
- [x] 更新 `CreateCollection` 处理器中的角色类型比较（`CollectionRoleTypeAdmin` → `CollectionRoleTypeOwner`）
- [x] 确保路由配置中的中间件引用正确
- [x] 更新 `NewHandler` 函数签名，增加 collectionRBACStorage 和 collectionRBACService 参数

## 任务 6: 更新测试用例 (collection_roles_test.go + e2e.go)
- [x] 更新角色命名断言（`%sOwner`, `%sOperator`, `%sTourist`）
- [x] 更新角色类型常量引用
- [x] 更新 `hasAllPermissions` 验证逻辑以适配正式权限code匹配
- [x] 更新 e2e.go 中的角色类型键名（`admin` → `owner`）

## 任务 7: 验证编译和运行
- [x] 编译后端（`go build ./cmd/server/...`）
- [x] `go vet ./internal/... ./pkg/... ./cmd/...` 通过
- [x] 启动服务并运行E2E烟雾测试

# 任务依赖关系
- 所有任务已完成

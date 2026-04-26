# 集合与RBAC模块全面重构 规范

## 为什么

当前的集合级RBAC系统存在以下问题：
1. 代码库缺少 `collection_rbac_storage.go` 文件导致编译失败
2. 集合权限以纯字符串常量存储在 `collection_role.permission_ids` 中，未在 `permissions` 集合中创建正式权限文档
3. 角色命名体系不够清晰（admin/data_admin/user），需要更业务化的命名（Owner/Operator/Tourist）
4. 权限检查中间件与业务代码混用系统级权限检查，导致集合级权限判断不够彻底

## 变更内容

### 核心重构

1. **恢复并重构 `collection_rbac_storage.go`** - 将 `CollectionRBACStorage` 接口和实现移到独立文件，修复编译错误
2. **创建集合时自动在 `permissions` 集合中创建正式权限文档** - 每创建一个集合，在 `rbac.permissions` 集合中创建5条权限记录：
   - `{module}:read` - 读取权限
   - `{module}:write` - 写入权限
   - `{module}:delete` - 删除权限
   - `{module}:admin` - 管理权限（包含字段管理）
   - `{module}:field:admin` - 字段管理权限
3. **重构角色命名体系**：
   - `{module}_admin` → `{module}Owner`（集合管理员）
   - `{module}_data_admin` → `{module}Operator`（数据操作员）
   - `{module}_user` → `{module}Tourist`（普通用户）
4. **角色类型常量重命名**：
   - `CollectionRoleTypeAdmin` → `CollectionRoleTypeOwner`
   - `CollectionRoleTypeDataAdmin` → `CollectionRoleTypeOperator`
   - `CollectionRoleTypeUser` → `CollectionRoleTypeTourist`

### 权限与角色映射关系

| 角色 | 代码 | 包含权限 | 说明 |
|------|------|----------|------|
| 集合管理员 | `{module}Owner` | `{module}:admin`, `read`, `write`, `delete`, `field:admin` | 集合完全控制，可管理字段 |
| 数据操作员 | `{module}Operator` | `{module}:read`, `write`, `delete` | 数据CRUD，不能管理字段 |
| 普通用户 | `{module}Tourist` | `{module}:read` | 只读访问 |

### 权限中间件重构

- `CollectionPermissionMiddleware` - 从URL参数 `:module` 提取模块并检查集合级权限
- `CollectionPermissionMiddlewareFromBody` - 从POST请求体提取 `module` 字段并检查权限
- `CollectionPermissionFieldAdminMiddleware` - 通过字段ID查找所属模块并检查 `field:admin` 权限
- 所有中间件同时支持 `system:admin` 超级管理员绕过检查

### InitDefaultData 增强

在 `rbac_storage.go` 的 `InitDefaultData()` 中增加MongoDB内置测试数据种子：
- 创建超级管理员角色 `root`（包含所有系统级权限）
- 创建管理员用户 `admin`（密码：`liangminchuan`）并关联 root 角色
- 自动创建系统级基础权限（user, role, permission, collection, field, data, scrape 各三级权限）

## 影响范围

- **受影响的规范**：集合管理、RBAC权限模型、业务数据API
- **受影响的核心代码**：
  - `internal/models/models.go` - 角色类型常量重命名
  - `pkg/rbac/collection_rbac.go` - 角色模板、权限检查逻辑重构
  - `internal/api/collection_permission_middleware.go` - 中间件保持不变，但权限字符串变更
  - `internal/api/handlers.go` - 创建集合处理器、路由配置
  - `internal/storage/rbac_storage.go` - InitDefaultData 增强
  - `internal/storage/collection_rbac_storage.go` - **新增文件**（恢复+重构）
  - `cmd/server/main.go` - 服务初始化流程微调
  - `test/collection_roles_test.go` - 测试用例适配新命名

## 新增需求

### 需求：集合创建时自动创建权限文档

系统**必须**在创建集合时在 `rbac.permissions` 集合中创建5个对应的权限文档。

#### 场景：创建权限成功
- **当** 用户创建新集合 `{module}`
- **则** 系统在 `permissions` 集合中创建以下记录：
  - `name`: "{module} 读取", `code`: "{module}:read"
  - `name`: "{module} 写入", `code`: "{module}:write"
  - `name`: "{module} 删除", `code`: "{module}:delete"
  - `name`: "{module} 管理", `code`: "{module}:admin"
  - `name`: "{module} 字段管理", `code`: "{module}:field:admin"

#### 场景：创建角色成功
- **当** 权限创建成功后
- **则** 系统创建3个集合角色，每个角色引用对应的权限ID

### 需求：权限检查中间件

系统**必须**在模块级API操作前检查用户是否具有对应集合权限。

#### 场景：URL模块参数权限检查
- **当** 用户请求 `GET /api/business/module/{module}`
- **则** 中间件提取 `module` 参数，检查用户是否拥有 `{module}:read` 权限
- **且** 无权限时返回HTTP 403

#### 场景：POST请求体权限检查
- **当** 用户请求 `POST /api/business`（module在请求体内）
- **则** 中间件从请求体提取 `module` 参数并检查写入权限

## 修改的需求

### 需求：角色类型常量

**修改前**：
```go
CollectionRoleTypeAdmin = "admin"
CollectionRoleTypeDataAdmin = "data_admin"
CollectionRoleTypeOperator = "operator"
CollectionRoleTypeUser = "user"
```

**修改后**：
```go
CollectionRoleTypeOwner = "owner"
CollectionRoleTypeOperator = "operator"
CollectionRoleTypeTourist = "tourist"
```

### 需求：角色创建模板

**修改前**：admin/data_admin/user 三个模板

**修改后**：owner/operator/tourist 三个模板，权限映射使用正式权限ID而非纯字符串

## 删除的需求

### 需求：旧的 operator 角色类型

**原因**：`operator` 角色类型未被使用，被 `data_admin` 替代。重构后统一为 `operator` 角色但包含数据管理员职责。

**迁移**：`data_admin` 类型直接重命名为 `operator`，功能不变。

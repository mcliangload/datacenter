# RBAC 权限模块 - 需求设计文档

## 1. 需求背景

数据中心系统面向多角色用户（系统管理员、数据操作员、只读用户），需要对系统资源进行细粒度的访问控制。采用 RBAC (Role-Based Access Control) 模型实现权限的灵活配置和管理。

## 2. 功能需求

### FR-RBAC-01: 用户管理

管理员可以对用户进行完整的生命周期管理。

| 操作 | 所需权限 |
|------|----------|
| 查看用户列表/详情 | user:read |
| 创建用户 | user:write |
| 更新用户信息 | user:write |
| 删除用户 | user:write |
| 为用户分配角色 | user:write |
| 移除用户角色 | user:write |
| 查看用户角色 | user:read |

### FR-RBAC-02: 角色管理

| 操作 | 所需权限 |
|------|----------|
| 查看角色列表/详情 | role:read |
| 创建角色 | role:write |
| 更新角色 | role:write |
| 删除角色 | role:write |
| 为角色分配权限 | role:write |
| 移除角色权限 | role:write |
| 查看角色权限 | role:read |

### FR-RBAC-03: 权限管理

| 操作 | 所需权限 |
|------|----------|
| 查看权限列表/详情 | permission:read |
| 创建权限 | permission:write |
| 更新权限 | permission:write |
| 删除权限 | permission:write |

### FR-RBAC-04: 权限检查

系统在每次 API 请求时检查用户是否拥有访问该资源所需的权限。

**检查流程**:
1. 获取当前登录用户
2. 遍历用户所有角色
3. 遍历角色所有权限
4. 匹配所需权限（精确匹配或通配符匹配）

### FR-RBAC-05: 超级管理员

拥有 `system:admin` 权限的用户可以绕过所有权限检查，访问系统全部资源。

### FR-RBAC-06: 通配符权限

权限代码支持通配符语法：`resource:*` 可以匹配该资源下的所有操作权限。

例如：
- `user:*` 匹配 `user:read`、`user:write`、`user:manage`
- `data:*` 匹配 `data:read`、`data:write`、`data:manage`

## 3. 权限代码规范

### 3.1 格式

```
<resource>:<action>
```

- `resource`: 资源名称（小写英文，如 user、role、data）
- `action`: 操作类型（read、write、manage 或 *）

### 3.2 内置权限代码

| 代码 | 含义 | 范围 |
|------|------|------|
| `user:read` | 查看用户 | 用户管理模块 |
| `user:write` | 管理用户（增删改） | 用户管理模块 |
| `user:manage` | 用户完全控制 | 用户管理模块 |
| `role:read` | 查看角色 | 角色管理模块 |
| `role:write` | 管理角色 | 角色管理模块 |
| `permission:read` | 查看权限 | 权限管理模块 |
| `permission:write` | 管理权限 | 权限管理模块 |
| `data:read` | 查看业务数据 | 业务数据模块 |
| `data:write` | 管理业务数据 | 业务数据模块 |
| `field:read` | 查看字段定义 | 字段管理模块 |
| `field:write` | 管理字段定义 | 字段管理模块 |
| `scrape:read` | 查看刮削任务 | 刮削管理模块 |
| `scrape:write` | 管理刮削任务 | 刮削管理模块 |
| `collection:read` | 查看集合 | 集合管理模块 |
| `collection:write` | 管理集合 | 集合管理模块 |
| `system:admin` | 超级管理员 | 全局 |

## 4. 数据关系设计

### 4.1 User-Role 关系

- 一个用户可以拥有多个角色
- 一个角色可以分配给多个用户
- 关系存储方式：User.role_ids 数组

### 4.2 Role-Permission 关系

- 一个角色可以包含多个权限
- 一个权限可以属于多个角色
- 关系存储方式：Role.permission_ids 数组

### 4.3 示例配置

```
角色: "数据操作员" (code: "data-operator")
  └── 权限: data:read, data:write, scrape:read, field:read

角色: "只读用户" (code: "viewer")
  └── 权限: data:read, scrape:read, collection:read, field:read
```

## 5. 非功能需求

| 编号 | 需求 | 说明 |
|------|------|------|
| NFR-RBAC-01 | 权限检查性能 | 单次检查在毫秒级完成 |
| NFR-RBAC-02 | 热更新 | 角色/权限变更后即时生效 |
| NFR-RBAC-03 | 数据一致性 | 删除角色后，用户不应保留对已删除角色的引用 |
| NFR-RBAC-04 | 最小权限原则 | 默认新用户无任何权限 |

## 6. 业务流程

### 6.1 权限检查流程

```
用户请求 GET /api/users
      │
      ▼
JWT 验证通过 → user_id 注入 ctx
      │
      ▼
PermissionMiddleware("user:read")
      │
      ▼
CheckPermission(userID, "user:read"):
  ├── 获取 user.RoleIDs = ["role_admin", "role_viewer"]
  ├── 遍历角色:
  │   ├── role("role_admin"):
  │   │   └── permissions: ["user:read", "user:write", "system:admin"]
  │   │       └── "system:admin" → 直接返回 true ✓
  │   └── (不继续检查)
  └── 返回 true
      │
      ▼
允许访问 → 执行 GetUsers handler
```

### 6.2 创建用户并分配角色

```
管理员 POST /api/users
  { username, password, email, role_ids: ["role_id_1"] }
      │
      ▼
JWT 验证 + user:write 权限检查
      │
      ▼
bcrypt 哈希密码
      │
      ▼
插入 users 集合 { username, password: hash, email, role_ids: [...] }
      │
      ▼
201 Created
```

## 7. 接口定义

### 用户管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/users | user:read | 用户列表（分页） |
| GET | /api/users/:id | user:read | 用户详情 |
| POST | /api/users | user:write | 创建用户 |
| PUT | /api/users/:id | user:write | 更新用户 |
| DELETE | /api/users/:id | user:write | 删除用户 |
| POST | /api/users/:id/roles | user:write | 分配角色 |
| DELETE | /api/users/:id/roles/:roleId | user:write | 移除角色 |
| GET | /api/users/:id/roles | user:read | 查看角色 |

### 角色管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/roles | role:read | 角色列表 |
| GET | /api/roles/:id | role:read | 角色详情 |
| POST | /api/roles | role:write | 创建角色 |
| PUT | /api/roles/:id | role:write | 更新角色 |
| DELETE | /api/roles/:id | role:write | 删除角色 |
| POST | /api/roles/:id/permissions | role:write | 分配权限 |
| DELETE | /api/roles/:id/permissions/:pid | role:write | 移除权限 |
| GET | /api/roles/:id/permissions | role:read | 查看权限 |

### 权限管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/permissions | permission:read | 权限列表 |
| GET | /api/permissions/:id | permission:read | 权限详情 |
| POST | /api/permissions | permission:write | 创建权限 |
| PUT | /api/permissions/:id | permission:write | 更新权限 |
| DELETE | /api/permissions/:id | permission:write | 删除权限 |

## 8. 默认数据

系统初始化时会创建以下默认数据：

### 默认权限
- user:read, user:write, user:manage
- role:read, role:write, role:manage
- permission:read, permission:write, permission:manage
- data:read, data:write, data:manage
- field:read, field:write, field:manage
- scrape:read, scrape:write, scrape:manage
- collection:read, collection:write, collection:manage
- system:admin

### 默认角色
- **管理员** (code: `admin`): 拥有 system:admin
- **普通用户** (code: `user`): 拥有 data:read, scrape:read

### 默认用户
- **admin**: 密码哈希存储，分配 admin 角色

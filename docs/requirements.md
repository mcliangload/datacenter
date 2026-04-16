# 数据中心系统需求设计文档

## 1. 各模块详细需求规格

### 1.1 用户权限系统 (RBAC)

#### 1.1.1 角色定义
- **root用户**：系统最高权限，可管理所有数据库和用户
- **datatypeowner**：数据库类型所有者，具备以下权限：
  - 定义所属database中datacollection的自定义字段
  - 对任意用户授予该database的dataowner权限
- **dataowner**：数据所有者，具备以下权限：
  - 对所属database的数据进行增删改查操作
  - 只能使用datatypeowner定义的key添加自定义字段

#### 1.1.2 权限管理
- 权限系统需嵌入完整审计(audit)数据，记录所有权限变更操作
- 支持角色分配和权限检查
- 实现权限验证中间件，保护API端点
- 权限和角色数据持久化存储在MongoDB的user数据库中
- 支持创建自定义角色和权限
- 支持角色与用户的绑定和解绑
- 支持角色与权限的绑定和解绑

### 1.2 日志系统

#### 1.2.1 日志等级
- 支持四个日志等级：DEBUG、INFO、WARN、ERROR
- 每个等级需支持三种输入类型：
  1. 纯字符串输入
  2. 带格式化占位符的字符串输入(如"name is %s", name)
  3. 自定义JSON字符串输入

#### 1.2.2 Gin日志中间件
- 记录以下请求信息：
  - 调用者身份
  - 方法名
  - 请求路径
  - 响应内容
  - 请求处理时间

### 1.3 业务数据管理

#### 1.3.1 数据模型
- 业务数据按模块划分，至少包含Model模块
- 每个模块包含三个collection：
  1. **自定义字段定义集合**：存储字段定义及其约束条件
  2. **业务数据集合**：存储实际业务数据
  3. **业务删除集合**：存储已删除数据，支持数据恢复功能

#### 1.3.2 数据删除策略
- 删除数据暂存于业务删除集合
- 数据删除后48小时自动完全清除

### 1.4 认证与授权

#### 1.4.1 JWT Token
- 实现基于JWT Token的前后端长连接认证机制
- Token需包含以下信息：
  - 用户身份标识
  - 用户角色信息
  - 权限范围
- 实现Token刷新机制和过期策略

### 1.5 数据模型设计

#### 1.5.1 基础结构
- 业务数据使用MongoDB自带的_id作为主键(PK)
- 所有业务数据需包含审计(audit)字段，记录：
  - 创建者(created_by)
  - 创建时间(created_at)
  - 更新者(updated_by)
  - 更新时间(updated_at)
- 业务数据基本结构：
  - 描述信息(description)
  - 自定义字段(custom_fields)
  - 文件路径(file_path)：仅需file_path和模块名即可创建数据

### 1.6 查询功能

#### 1.6.1 JQL查询
- 实现类JIRA JQL查询语句解析器，支持"字段 运算符 值/函数"格式
- 支持以下内置函数：
  - CurrentUser()：获取当前用户
  - 时间相关函数(如Now(), StartOfDay(), EndOfWeek()等)
- 提供完整的JQL语法支持文档
- 实现JQL到MongoDB查询语句的转换逻辑
- 提供查询性能优化方案

## 2. 功能点描述及验收标准

### 2.1 用户管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 用户注册 | 创建新用户，设置用户名、密码和角色 | 成功创建用户，返回用户信息 |
| 用户登录 | 验证用户名和密码，生成JWT Token | 成功登录，返回Token和用户信息 |
| 用户列表 | 获取用户列表，支持分页和筛选 | 正确返回用户列表，支持分页 |
| 用户详情 | 获取单个用户详细信息 | 正确返回用户详情 |
| 更新用户 | 更新用户信息，包括角色和权限 | 成功更新用户信息 |
| 删除用户 | 删除指定用户 | 成功删除用户 |
| 分配角色 | 为用户分配角色 | 成功分配角色，用户拥有角色权限 |
| 移除角色 | 从用户移除角色 | 成功移除角色，用户不再拥有角色权限 |
| 获取用户角色 | 获取用户的角色列表 | 正确返回用户的角色列表 |

### 2.2 权限管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建权限 | 创建新的权限 | 成功创建权限，返回权限信息 |
| 权限列表 | 获取权限列表，支持分页 | 正确返回权限列表，支持分页 |
| 权限详情 | 获取单个权限详细信息 | 正确返回权限详情 |
| 更新权限 | 更新权限信息 | 成功更新权限信息 |
| 删除权限 | 删除指定权限 | 成功删除权限 |

### 2.3 角色管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建角色 | 创建新的角色，设置名称和权限 | 成功创建角色，返回角色信息 |
| 角色列表 | 获取角色列表，支持分页 | 正确返回角色列表，支持分页 |
| 角色详情 | 获取单个角色详细信息 | 正确返回角色详情 |
| 更新角色 | 更新角色信息，包括名称和权限 | 成功更新角色信息 |
| 删除角色 | 删除指定角色 | 成功删除角色 |
| 分配权限 | 为角色分配权限 | 成功分配权限，角色拥有权限 |
| 移除权限 | 从角色移除权限 | 成功移除权限，角色不再拥有权限 |
| 获取角色权限 | 获取角色的权限列表 | 正确返回角色的权限列表 |

### 2.4 字段定义管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建字段定义 | 为模块创建新的字段定义，包括字段类型和约束 | 成功创建字段定义 |
| 获取字段定义列表 | 获取指定模块的字段定义列表 | 正确返回字段定义列表 |
| 更新字段定义 | 更新现有字段定义 | 成功更新字段定义 |
| 删除字段定义 | 删除指定字段定义 | 成功删除字段定义 |

### 2.5 业务数据管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建业务数据 | 创建新的业务数据，支持自定义字段 | 成功创建业务数据 |
| 获取业务数据列表 | 获取指定模块的业务数据列表，支持JQL查询 | 正确返回业务数据列表，支持JQL查询 |
| 获取业务数据详情 | 获取单个业务数据详细信息 | 正确返回业务数据详情 |
| 更新业务数据 | 更新现有业务数据 | 成功更新业务数据 |
| 删除业务数据 | 软删除业务数据，暂存到删除集合 | 成功软删除业务数据 |

### 2.6 已删除数据管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 获取已删除数据列表 | 获取指定模块的已删除数据列表 | 正确返回已删除数据列表 |
| 恢复已删除数据 | 从删除集合恢复数据到业务数据集合 | 成功恢复已删除数据 |
| 清理过期数据 | 自动清理48小时前的已删除数据 | 成功清理过期数据 |

## 3. 接口定义规范

### 3.1 通用规范
- 所有API接口遵循RESTful设计风格
- 使用HTTP方法：GET(获取)、POST(创建)、PUT(更新)、DELETE(删除)
- 响应格式：
  ```json
  {
    "code": 200,
    "message": "success",
    "data": { ... }
  }
  ```
- 错误响应格式：
  ```json
  {
    "code": 400,
    "message": "error message"
  }
  ```

### 3.2 具体接口

#### 3.2.1 认证接口
- **POST /api/auth/login**：用户登录
  - 请求体：`{"username": "...", "password": "..."}`
  - 响应：Token和用户信息

#### 3.2.2 用户管理接口
- **POST /api/users**：创建用户
- **GET /api/users**：获取用户列表
- **GET /api/users/:id**：获取用户详情
- **PUT /api/users/:id**：更新用户
- **DELETE /api/users/:id**：删除用户
- **POST /api/users/:id/roles**：分配角色给用户
- **DELETE /api/users/:id/roles/:roleId**：从用户移除角色
- **GET /api/users/:id/roles**：获取用户的角色列表

#### 3.2.3 权限管理接口
- **POST /api/permissions**：创建权限
- **GET /api/permissions**：获取权限列表
- **GET /api/permissions/:id**：获取权限详情
- **PUT /api/permissions/:id**：更新权限
- **DELETE /api/permissions/:id**：删除权限

#### 3.2.4 角色管理接口
- **POST /api/roles**：创建角色
- **GET /api/roles**：获取角色列表
- **GET /api/roles/:id**：获取角色详情
- **PUT /api/roles/:id**：更新角色
- **DELETE /api/roles/:id**：删除角色
- **POST /api/roles/:id/permissions**：分配权限给角色
- **DELETE /api/roles/:id/permissions/:permissionId**：从角色移除权限
- **GET /api/roles/:id/permissions**：获取角色的权限列表

#### 3.2.5 字段定义接口
- **POST /api/fields**：创建字段定义
- **GET /api/fields/module/:module**：获取模块字段定义列表
- **GET /api/fields/:id**：获取字段定义详情
- **PUT /api/fields/:id**：更新字段定义
- **DELETE /api/fields/:id**：删除字段定义

#### 3.2.6 业务数据接口
- **POST /api/business**：创建业务数据
- **GET /api/business/module/:module**：获取模块业务数据列表，支持JQL查询
- **GET /api/business/:id**：获取业务数据详情
- **PUT /api/business/:id**：更新业务数据
- **DELETE /api/business/:id**：删除业务数据

#### 3.2.7 已删除数据接口
- **GET /api/deleted/module/:module**：获取模块已删除数据列表
- **GET /api/deleted/:id**：获取已删除数据详情
- **POST /api/deleted/:id/recover**：恢复已删除数据

## 4. 数据字典

### 4.1 用户表 (users)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 用户ID | 主键 |
| username | String | 用户名 | 唯一，必填 |
| password | String | 密码 | 加密存储，必填 |
| email | String | 邮箱 | 唯一，必填 |
| roles | Array<String> | 角色列表 | 必填 |
| permissions | Array<String> | 权限列表 | 自动生成 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 4.2 权限表 (permissions)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 权限ID | 主键 |
| name | String | 权限名称 | 必填 |
| code | String | 权限代码 | 唯一，必填 |
| description | String | 权限描述 | 可选 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 4.3 角色表 (roles)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 角色ID | 主键 |
| name | String | 角色名称 | 必填 |
| code | String | 角色代码 | 唯一，必填 |
| description | String | 角色描述 | 可选 |
| permissions | Array<String> | 权限列表 | 可选 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 4.4 用户角色关联表 (user_roles)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 关联ID | 主键 |
| user_id | String | 用户ID | 必填 |
| role_id | String | 角色ID | 必填 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 4.5 角色权限关联表 (role_permissions)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 关联ID | 主键 |
| role_id | String | 角色ID | 必填 |
| permission_id | String | 权限ID | 必填 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 4.6 字段定义表 (field_definitions)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 字段定义ID | 主键 |
| module | String | 模块名 | 必填 |
| field_name | String | 字段名 | 必填 |
| field_type | String | 字段类型 | 必填 (int/float/string/list) |
| description | String | 字段描述 | 可选 |
| constraints | Object | 字段约束 | 可选 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 4.7 业务数据表 (business_data)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 数据ID | 主键 |
| module | String | 模块名 | 必填 |
| description | String | 描述信息 | 可选 |
| custom_fields | Object | 自定义字段 | 可选 |
| file_path | String | 文件路径 | 可选 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 4.8 已删除数据表 (deleted_data)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 删除记录ID | 主键 |
| module | String | 模块名 | 必填 |
| original_id | ObjectID | 原始数据ID | 必填 |
| description | String | 描述信息 | 可选 |
| custom_fields | Object | 自定义字段 | 可选 |
| file_path | String | 文件路径 | 可选 |
| deleted_at | DateTime | 删除时间 | 自动生成 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

## 5. 异常处理策略

### 5.1 错误分类

| 错误类型 | 状态码 | 描述 | 处理方式 |
|----------|--------|------|----------|
| 认证错误 | 401 | 未认证或Token无效 | 返回认证错误信息，引导用户重新登录 |
| 授权错误 | 403 | 无权限访问 | 返回授权错误信息，提示用户联系管理员 |
| 参数错误 | 400 | 请求参数无效 | 返回参数错误信息，提示用户修正参数 |
| 资源不存在 | 404 | 请求资源不存在 | 返回资源不存在错误信息 |
| 服务器错误 | 500 | 服务器内部错误 | 返回服务器错误信息，记录详细日志 |
| 数据库错误 | 500 | 数据库操作失败 | 返回数据库错误信息，记录详细日志 |

### 5.2 错误处理流程

1. 捕获错误：在各个层级捕获可能的错误
2. 记录错误：使用日志系统记录错误详情
3. 转换错误：将内部错误转换为用户友好的错误信息
4. 返回错误：根据错误类型返回相应的HTTP状态码和错误信息

### 5.3 异常日志

- 记录异常发生的时间、位置、原因和上下文信息
- 对于关键操作的异常，发送告警通知
- 定期分析异常日志，优化系统稳定性

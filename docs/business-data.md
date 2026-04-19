# 业务数据管理

## 1. 数据上传与处理流程

### 1.1 上传参数

用户上传时需提供三个关键参数：
| 参数 | 类型 | 描述 |
|------|------|------|
| data_path | String | 数据文件路径 |
| scraper_path | String | 刮削器脚本路径 |
| module | String | 模块名 |

### 1.2 处理流程（并发刮削）

```
1. 用户上传：POST /api/scraper/upload
   请求体: {"data_path": "xxx", "scraper_path": "xxx", "module": "xxx"}

2. 系统验证:
   - 验证模块是否存在（查询field_definitions）
   - 验证数据路径是否有效
   - 验证刮削器路径是否存在
   - 验证用户是否有该模块的创建权限（通过RBAC）

3. 创建刮削任务记录:
   - 在scrape_tasks集合中创建任务记录
   - 状态设为"刮削中"(scraping)
   - 记录创建者、创建时间等审计信息

4. 异步启动刮削处理:
   - 将刮削任务提交到后台处理队列
   - 立即返回任务ID和启动成功消息给用户

5. 后台刮削处理:
   - 执行刮削器脚本
   - 传入数据路径作为参数
   - 刮削器返回处理结果
   - 记录刮削执行日志

6. 结果处理:
   - 刮削成功：更新任务状态为"成功"(success)，存储结果至模块集合
   - 刮削失败：更新任务状态为"失败"(failed)，记录错误信息

7. 任务状态查询:
   - 用户通过 GET /api/scraper/tasks/:id 查询刮削状态
   - 系统返回当前状态和处理结果
```

### 1.3 刮削重试机制

- 刮削失败的数据保留在scrape_tasks集合中
- 用户可重新上传刮削器并发起重试
- 重试时传入相同的data_path和新的scraper_path
- 系统创建新的刮削任务记录

## 2. 刮削状态管理

### 2.1 状态集合

使用 `scrape_tasks` 集合存储刮削任务状态：

| 状态 | 描述 |
|------|------|
| scraping | 刮削中 |
| success | 刮削成功 |
| failed | 刮削失败 |

### 2.2 状态流转

```
[新建任务] → [scraping] → [success] (保留记录)
                    ↓
              [failed] → [等待重试] → [scraping]
```

### 2.3 任务管理

- 刮削任务记录永久保留，不自动删除
- 提供任务清理API，允许手动删除旧任务
- 支持按状态、模块、时间等条件查询任务

## 3. 数据集合管理

### 3.1 多集合支持

系统支持将刮削结果存储至多个不同集合：
- 每个模块对应一个独立的MongoDB集合
- 集合名称格式：`{module}_data`
- 例如：`movie_data`、`book_data`、`music_data`

### 3.2 动态集合创建

- 支持在运行时动态创建新的集合
- 当模块首次上传数据时，自动创建对应的集合
- 集合创建时自动配置索引和权限

### 3.3 datatypeowner职责

- 每个集合必须由唯一的datatypeowner进行定义和创建
- datatypeowner负责：
  - 定义集合的字段结构（field_definitions）
  - 创建对应集合的角色（通过RBAC）
  - 为其他用户授予该集合的角色（通过RBAC）
  - 管理集合的元数据和索引

### 3.4 集合创建流程

```
1. datatypeowner调用 POST /api/collections
   请求体: {"module": "movie", "description": "电影数据"}

2. 系统验证:
   - 验证调用者是否为datatypeowner（通过RBAC）
   - 验证模块名是否已存在

3. 创建模块记录:
   - 在collections集合中创建模块元数据
   - 初始化对应的数据集合
   - 配置集合索引

4. 自动创建对应集合的角色:
   - 创建 `{module}_admin` 角色（完全控制权限）
   - 创建 `{module}_user` 角色（基础操作权限）
   - 创建 `{module}_viewer` 角色（只读权限）
   - datatypeowner自动获得 `{module}_admin` 角色

5. 授予datatypeowner该集合的全部权限
6. 记录集合创建日志
```

## 4. 权限管理（通过RBAC实现）

### 4.1 权限类型

| 权限 | 代码 | 描述 |
|------|------|------|
| 读取 | read | 查看集合数据 |
| 创建 | create | 向集合添加数据 |
| 更新 | update | 修改集合数据 |
| 删除 | delete | 删除集合数据 |
| 管理 | admin | 管理集合权限和角色 |

### 4.2 角色管理

每个集合自动创建以下角色：

| 角色 | 代码 | 权限 | 描述 |
|------|------|------|------|
| 集合管理员 | {module}_admin | read,create,update,delete,admin | 完全控制权限 |
| 集合用户 | {module}_user | read,create,update | 基础操作权限 |
| 集合只读 | {module}_viewer | read | 只读权限 |

### 4.3 权限授予流程

```
1. datatypeowner调用 POST /api/roles/{roleId}/users
   请求体: {"user_id": "xxx"}

2. 系统验证:
   - 验证调用者是否为datatypeowner（通过RBAC）
   - 验证目标用户是否存在
   - 验证角色是否存在且属于该集合

3. 存储角色分配:
   - 在用户的role_ids数组中添加角色ID

4. 记录权限授予日志
5. 返回授权结果
```

### 4.4 权限检查

- 所有数据操作前检查用户权限（通过RBAC）
- 无权限用户返回403 Forbidden
- 权限检查通过用户-角色-权限的三层结构实现

## 5. 功能点及验收标准

### 5.1 数据上传与刮削

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 上传刮削任务 | 接收数据路径、刮削器路径和模块名 | 成功创建刮削任务，返回任务ID |
| 验证模块存在 | 检查指定模块是否已定义 | 模块存在返回true，不存在返回错误 |
| 并发刮削处理 | 后台异步执行刮削器 | 立即返回成功，刮削在后台执行 |
| 存储刮削结果 | 将刮削结果存储至MongoDB | 成功存储并更新任务状态 |
| 任务状态查询 | 查询刮削任务的当前状态 | 正确返回状态和处理结果 |

### 5.2 刮削状态管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 查看刮削状态 | 获取任务的当前刮削状态 | 正确返回状态（scraping/success/failed） |
| 重试刮削 | 使用新刮削器重新处理失败任务 | 成功重新执行刮削 |
| 获取失败任务列表 | 获取所有刮削失败的任务 | 返回失败任务列表及错误信息 |
| 清理任务 | 手动清理旧的刮削任务 | 成功删除指定任务 |

### 5.3 集合管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建集合 | datatypeowner创建新的数据集合 | 成功创建并返回集合信息 |
| 动态集合创建 | 首次上传数据时自动创建集合 | 成功创建集合并配置索引 |
| 查看集合列表 | 获取所有可用集合 | 返回集合列表及元数据 |
| 设置datatypeowner | 为集合指定所有者 | 成功设置，仅一人可成为所有者 |
| 转移所有权 | datatypeowner可将所有权转移给他人 | 成功转移，新owner获得全部权限 |

### 5.4 角色与权限管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建集合角色 | 自动为集合创建对应的角色 | 成功创建角色并分配默认权限 |
| 授予角色 | datatypeowner为用户授予集合角色 | 成功授予，用户获得对应权限 |
| 撤销角色 | datatypeowner撤销用户的集合角色 | 成功撤销，用户失去对应权限 |
| 查看角色 | 查看集合的所有角色配置 | 返回角色列表 |
| 权限检查 | 数据操作前验证用户权限 | 正确返回是否有权限 |

### 5.5 字段定义管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建字段定义 | 为模块创建新的字段定义 | 成功创建字段定义 |
| 获取字段定义列表 | 获取指定模块的字段定义列表 | 正确返回字段定义列表 |
| 更新字段定义 | 更新现有字段定义 | 成功更新字段定义 |
| 删除字段定义 | 删除指定字段定义 | 成功删除字段定义 |

### 5.6 业务数据管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建业务数据 | 通过接口直接输入属性创建数据 | 成功创建业务数据 |
| 上传刮削数据 | 通过刮削器处理并存储数据 | 成功存储刮削结果 |
| 获取业务数据列表 | 获取指定模块的业务数据列表 | 正确返回业务数据列表 |
| 获取业务数据详情 | 获取单个业务数据详细信息 | 正确返回业务数据详情 |
| 更新业务数据 | 更新现有业务数据 | 成功更新业务数据 |
| 删除业务数据 | 软删除业务数据 | 成功软删除业务数据 |

## 6. API 接口

### 6.1 刮削相关接口

- **POST /api/scraper/upload**：上传刮削任务（异步处理）
- **GET /api/scraper/tasks**：获取刮削任务列表
- **GET /api/scraper/tasks/:id**：获取任务详情
- **POST /api/scraper/tasks/:id/retry**：重试失败任务
- **DELETE /api/scraper/tasks/:id**：删除任务

### 6.2 集合管理接口

- **POST /api/collections**：创建集合
- **GET /api/collections**：获取集合列表
- **GET /api/collections/:module**：获取集合详情
- **PUT /api/collections/:module**：更新集合信息
- **DELETE /api/collections/:module**：删除集合
- **POST /api/collections/:module/indexes**：创建索引
- **GET /api/collections/:module/indexes**：查看索引列表
- **DELETE /api/collections/:module/indexes/:name**：删除索引

### 6.3 业务数据接口

- **POST /api/business**：创建业务数据（直接输入属性）
- **GET /api/business/module/:module**：获取模块业务数据列表
- **GET /api/business/:id**：获取业务数据详情
- **PUT /api/business/:id**：更新业务数据
- **DELETE /api/business/:id**：删除业务数据

### 6.4 字段定义接口

- **POST /api/fields**：创建字段定义
- **GET /api/fields/module/:module**：获取模块字段定义列表
- **GET /api/fields/:id**：获取字段定义详情
- **PUT /api/fields/:id**：更新字段定义
- **DELETE /api/fields/:id**：删除字段定义

## 7. 数据模型

### 7.1 刮削任务表 (scrape_tasks)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 任务ID | 主键 |
| module | String | 模块名 | 必填 |
| data_path | String | 数据文件路径 | 必填 |
| scraper_path | String | 刮削器路径 | 必填 |
| status | String | 刮削状态 | scraping/success/failed |
| result | Object | 刮削结果 | 可选 |
| error_message | String | 错误信息 | 刮削失败时填写 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_at | DateTime | 更新时间 | 自动生成 |
| started_at | DateTime | 开始刮削时间 | 可选 |
| completed_at | DateTime | 完成刮削时间 | 可选 |

### 7.2 集合元数据表 (collections)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 集合ID | 主键 |
| module | String | 模块名 | 唯一，必填 |
| description | String | 集合描述 | 可选 |
| datatypeowner | String | 数据类型所有者ID | 必填 |
| collection_name | String | MongoDB集合名称 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 7.3 字段定义表 (field_definitions)

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

### 7.4 业务数据表 (business_data)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 数据ID | 主键 |
| module | String | 模块名 | 必填 |
| description | String | 描述信息 | 可选 |
| custom_fields | Object | 自定义字段（JSON格式） | 可选 |
| file_path | String | 文件路径 | 可选 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

## 8. 索引策略

### 8.1 集合索引

为提高查询性能，系统自动为各集合创建以下索引：

| 集合 | 索引字段 | 类型 | 描述 |
|------|----------|------|------|
| collections | module | 唯一索引 | 快速查找模块 |
| field_definitions | module | 普通索引 | 按模块查询字段定义 |
| business_data | module | 普通索引 | 按模块查询数据 |
| business_data | created_by | 普通索引 | 按创建者查询数据 |
| business_data | created_at | 普通索引 | 按时间排序 |
| scrape_tasks | module | 普通索引 | 按模块查询任务 |
| scrape_tasks | status | 普通索引 | 按状态查询任务 |
| scrape_tasks | created_at | 普通索引 | 按时间查询任务 |

### 8.2 动态索引

由于MongoDB支持动态模式，刮削器的结果可能包含各种不同的字段，系统支持：

- **自动索引建议**：基于查询模式自动建议索引
- **手动索引管理**：datatypeowner可通过接口为特定字段创建索引
- **复合索引**：支持创建多字段复合索引提高复杂查询性能

### 8.3 索引管理API

- **POST /api/collections/:module/indexes**：创建索引
- **GET /api/collections/:module/indexes**：查看索引列表
- **DELETE /api/collections/:module/indexes/:name**：删除索引

## 9. 日志管理

### 9.1 日志类型

| 日志类型 | 描述 | 存储位置 |
|----------|------|----------|
| 操作日志 | 用户操作记录 | logs/operation.log |
| 刮削日志 | 刮削器执行记录 | logs/scraper.log |
| 错误日志 | 系统错误记录 | logs/error.log |
| 审计日志 | 权限变更记录 | logs/audit.log |

### 9.2 日志内容

#### 刮削日志

```json
{
  "time": "2026-04-18T10:00:00Z",
  "level": "info",
  "task_id": "task123",
  "module": "movie",
  "data_path": "/path/to/data",
  "scraper_path": "/path/to/scraper.py",
  "status": "started",
  "message": "刮削任务开始执行"
}
```

#### 操作日志

```json
{
  "time": "2026-04-18T10:00:00Z",
  "level": "info",
  "user_id": "user123",
  "action": "create_collection",
  "module": "movie",
  "message": "创建电影数据集合"
}
```

### 9.3 日志记录点

系统在以下关键节点记录日志：

1. **刮削相关**
   - 任务创建
   - 任务开始执行
   - 任务执行完成（成功/失败）
   - 任务重试

2. **集合管理**
   - 集合创建
   - 集合更新
   - 集合删除
   - 索引管理操作

3. **权限管理**
   - 角色创建
   - 角色分配
   - 角色撤销
   - 权限变更

4. **数据操作**
   - 数据创建
   - 数据更新
   - 数据删除
   - 数据查询（重要操作）

## 10. 基础结构要求

### 10.1 主键设计

- 业务数据使用MongoDB自带的_id作为主键(PK)

### 10.2 审计字段

所有业务数据需包含审计(audit)字段，记录：
- 创建者(created_by)
- 创建时间(created_at)
- 更新者(updated_by)
- 更新时间(updated_at)

### 10.3 datatypeowner权限

- datatypeowner拥有集合的全部权限
- datatypeowner可以为其他用户授予集合对应的角色
- datatypeowner身份不可撤销，只能转移

### 10.4 刮削器规范

刮削器需符合以下规范：
- 支持命令行调用：`python scraper.py <data_path>`
- 返回JSON格式的处理结果
- 成功时返回：`{"success": true, "data": {...}}`
- 失败时返回：`{"success": false, "error": "错误信息"}`

### 10.5 数据存储特点

- 利用MongoDB的JSON格式存储，支持无限多种数据结构
- 刮削器结果直接以JSON形式存储在custom_fields字段中
- 支持嵌套结构、数组等复杂数据类型
- 无需预先定义所有字段结构，适应不同刮削器的输出

### 10.6 并发处理

- 刮削任务采用异步并发处理
- 使用工作队列管理刮削任务
- 支持配置刮削任务的并发度
- 任务状态实时更新

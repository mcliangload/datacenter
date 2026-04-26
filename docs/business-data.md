# 业务数据管理

## 1. 数据上传与处理流程

### 1.1 上传参数

用户上传时需提供三个关键参数：
| 参数 | 类型 | 描述 |
|------|------|------|
| module | String | 模块名（必填） |
| data_path | String | 数据文件路径（必填） |
| scraper_path | String | 刮削器脚本路径（必填） |

### 1.2 处理流程（并发刮削）

```
1. 用户上传：POST /api/scraper/upload
   请求体: {"module": "movie", "data_path": "/data/movies.json", "scraper_path": "/scrapers/movie_scraper.py"}

2. 系统验证:
   - 验证模块是否存在（查询collections集合）
   - 验证数据路径是否有效（文件存在性检查）
   - 验证刮削器路径是否存在（文件存在性检查）
   - 验证用户是否有该模块的刮削权限（通过RBAC）

3. 创建刮削任务记录:
   - 在scrape_tasks集合中创建任务记录
   - 状态设为"pending"(等待处理)
   - 记录创建者、创建时间等审计信息

4. 异步启动刮削处理:
   - 将刮削任务提交到后台任务队列
   - 立即返回任务ID和提交成功消息给用户

5. 后台刮削处理:
   - 工作协程从队列取出任务
   - 更新状态为"scraping"(刮削中)
   - 执行刮削器脚本：`python {scraper_path} {data_path}`
   - 刮削器返回JSON格式处理结果
   - 记录刮削执行日志

6. 结果处理:
   - 刮削成功：更新任务状态为"success"，存储结果至 {module}_data 集合
   - 刮削失败：更新任务状态为"failed"，记录错误信息

7. 任务状态查询:
   - 用户通过 GET /api/scraper/tasks/:id 查询刮削状态
   - 系统返回当前状态和处理结果
```

### 1.3 刮削器规范

刮削器需符合以下规范：
- 支持命令行调用：`python {scraper_path} {data_path}`
- 返回JSON格式的处理结果
- 成功时返回：`{"success": true, "data": {...}}`
- 失败时返回：`{"success": false, "error": "错误信息"}`

### 1.4 刮削重试机制

- 刮削失败的任务可以重试
- 重试时传入相同的data_path和新的scraper_path
- 系统创建新的刮削任务记录

## 2. 刮削状态管理

### 2.1 状态集合

使用 `scrape_tasks` 集合存储刮削任务状态：

| 状态 | 描述 |
|------|------|
| pending | 等待处理 |
| scraping | 刮削中 |
| success | 刮削成功 |
| failed | 刮削失败 |

### 2.2 状态流转

```
[新建任务] → [pending] → [scraping] → [success] (保留记录)
                              ↓
                        [failed] → [重新提交] → [pending]
```

### 2.3 任务管理

- 刮削任务记录支持软删除，移至deleted_scrape_tasks集合
- 提供任务恢复API，从deleted_scrape_tasks恢复到scrape_tasks
- 支持按状态、模块、时间等条件查询任务
- 支持批量删除任务

## 3. 数据集合管理

### 3.1 多集合支持

系统支持将刮削结果存储至多个不同集合：
- 每个模块对应一个独立的MongoDB集合
- 集合名称格式：`{module}_data`
- 例如：`movie_data`、`book_data`、`music_data`

### 3.2 动态集合创建

- 支持在运行时动态创建新的集合
- 当创建集合时，自动创建对应的数据集合
- 集合创建时自动配置索引

### 3.3 datatypeowner职责

- 每个集合有一个datatypeowner进行管理
- datatypeowner负责：
  - 定义集合的字段结构（field_definitions）
  - 管理集合的元数据和索引
  - 设置集合的所有者

### 3.4 集合创建流程

```
1. datatypeowner调用 POST /api/collections
   请求体: {"module": "movie", "description": "电影数据"}

2. 系统验证:
   - 验证调用者是否有集合创建权限（通过RBAC）
   - 验证模块名是否已存在

3. 创建模块记录:
   - 在collections集合中创建模块元数据
   - 初始化对应的数据集合
   - 配置集合索引

4. 记录集合创建日志
```

## 4. 权限管理（通过RBAC实现）

### 4.1 权限类型

系统使用基于资源的权限代码格式：

| 权限代码 | 描述 |
|------|------|
| user:* | 用户完全控制 |
| user:read | 查看用户 |
| user:write | 管理用户 |
| role:* | 角色完全控制 |
| role:read | 查看角色 |
| role:write | 管理角色 |
| permission:* | 权限完全控制 |
| permission:read | 查看权限 |
| permission:write | 管理权限 |
| data:* | 数据完全控制 |
| data:read | 查看数据 |
| data:write | 管理数据 |
| field:* | 字段完全控制 |
| field:read | 查看字段 |
| field:write | 管理字段 |
| scrape:* | 刮削完全控制 |
| scrape:read | 查看刮削任务 |
| scrape:write | 管理刮削任务 |
| collection:* | 集合完全控制 |
| collection:read | 查看集合 |
| collection:write | 管理集合 |

### 4.2 权限检查

- 所有数据操作前检查用户权限（通过RBAC）
- 无权限用户返回403 Forbidden
- 权限检查支持通配符匹配（如 `data:*` 匹配所有数据操作权限）

## 5. 功能点及验收标准

### 5.1 数据上传与刮削

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 上传刮削任务 | 接收模块名、数据路径和刮削器路径 | 成功创建刮削任务，返回任务ID |
| 验证模块存在 | 检查指定模块是否已定义 | 模块存在返回正常，不存在返回错误 |
| 并发刮削处理 | 后台异步执行刮削器 | 立即返回成功，刮削在后台执行 |
| 存储刮削结果 | 将刮削结果存储至MongoDB | 成功存储并更新任务状态 |
| 任务状态查询 | 查询刮削任务的当前状态 | 正确返回状态（pending/scraping/success/failed） |

### 5.2 刮削状态管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 查看刮削状态 | 获取任务的当前刮削状态 | 正确返回状态 |
| 重试刮削 | 使用新刮削器重新处理失败任务 | 成功重新执行刮削 |
| 获取失败任务列表 | 获取所有刮削失败的任务 | 返回失败任务列表及错误信息 |
| 清理任务 | 手动清理旧的刮削任务 | 成功软删除指定任务 |
| 恢复任务 | 恢复已删除的刮削任务 | 成功从deleted_scrape_tasks恢复 |

### 5.3 集合管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建集合 | datatypeowner创建新的数据集合 | 成功创建并返回集合信息 |
| 查看集合列表 | 获取所有可用集合 | 返回集合列表及元数据 |
| 查看集合详情 | 获取指定模块的集合信息 | 返回集合详情 |
| 更新集合 | 更新集合信息 | 成功更新集合信息 |
| 删除集合 | 删除指定集合 | 成功删除集合及其数据 |
| 创建索引 | 为集合创建索引 | 成功创建索引 |
| 查看索引列表 | 获取集合的所有索引 | 返回索引列表 |
| 删除索引 | 删除指定索引 | 成功删除索引 |

### 5.4 字段定义管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建字段定义 | 为模块创建新的字段定义 | 成功创建字段定义 |
| 获取字段定义列表 | 获取指定模块的字段定义列表 | 正确返回字段定义列表 |
| 更新字段定义 | 更新现有字段定义 | 成功更新字段定义 |
| 删除字段定义 | 删除指定字段定义 | 成功删除字段定义 |

### 5.5 业务数据管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 创建业务数据 | 通过接口直接输入属性创建数据 | 成功创建业务数据 |
| 上传刮削数据 | 通过刮削器处理并存储数据 | 成功存储刮削结果 |
| 获取业务数据列表 | 获取指定模块的业务数据列表 | 正确返回业务数据列表，支持JQL查询 |
| 获取业务数据详情 | 获取单个业务数据详细信息 | 正确返回业务数据详情 |
| 更新业务数据 | 更新现有业务数据 | 成功更新业务数据 |
| 删除业务数据 | 软删除业务数据 | 成功软删除业务数据 |

### 5.6 已删除数据管理

| 功能点 | 描述 | 验收标准 |
|--------|------|----------|
| 获取已删除数据列表 | 获取指定模块的已删除数据 | 返回已删除数据列表 |
| 获取已删除数据详情 | 获取单个已删除数据详情 | 返回已删除数据详情 |
| 恢复已删除数据 | 恢复已删除的业务数据 | 成功恢复数据 |

## 6. API 接口

### 6.1 刮削相关接口

- **POST /api/scraper/upload**：提交刮削任务
- **GET /api/scraper/tasks**：获取刮削任务列表
- **GET /api/scraper/tasks/:id**：获取任务详情
- **POST /api/scraper/tasks/:id/retry**：重试失败任务
- **DELETE /api/scraper/tasks/:id**：删除任务（软删除）
- **POST /api/scraper/tasks/batch-delete**：批量删除任务

### 6.2 已删除刮削任务接口

- **GET /api/deleted-scraper/module/:module**：获取已删除任务列表（module=all时查询所有）
- **GET /api/deleted-scraper/:id**：获取已删除任务详情
- **POST /api/deleted-scraper/:id/recover**：恢复已删除任务

### 6.3 集合管理接口

- **POST /api/collections**：创建集合
- **GET /api/collections**：获取集合列表
- **GET /api/collections/:module**：获取集合详情
- **PUT /api/collections/:module**：更新集合信息
- **DELETE /api/collections/:module**：删除集合
- **POST /api/collections/:module/indexes**：创建索引
- **GET /api/collections/:module/indexes**：查看索引列表
- **DELETE /api/collections/:module/indexes/:name**：删除索引

### 6.4 业务数据接口

- **POST /api/business**：创建业务数据
- **GET /api/business/module/:module**：获取模块业务数据列表
- **GET /api/business/module/:module/:id**：获取业务数据详情
- **PUT /api/business/module/:module/:id**：更新业务数据
- **DELETE /api/business/module/:module/:id**：删除业务数据

### 6.5 已删除业务数据接口

- **GET /api/deleted/module/:module**：获取已删除数据列表
- **GET /api/deleted/:id**：获取已删除数据详情
- **POST /api/deleted/:id/recover**：恢复已删除数据

### 6.6 字段定义接口

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
| status | String | 刮削状态 | pending/scraping/success/failed |
| result | Object | 刮削结果 | 可选 |
| error_message | String | 错误信息 | 刮削失败时填写 |
| started_at | DateTime | 开始刮削时间 | 可选 |
| completed_at | DateTime | 完成刮削时间 | 可选 |
| business_data_id | ObjectID | 关联的业务数据ID | 可选 |
| description | String | 任务描述 | 可选 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 7.2 集合元数据表 (collections)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 集合ID | 主键 |
| module | String | 模块名 | 唯一，必填 |
| description | String | 集合描述 | 可选 |
| datatype_owner | String | 数据类型所有者ID | 必填 |
| collection_name | String | MongoDB集合名称 | 必填 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 7.3 字段定义表 (field_definitions)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 字段定义ID | 主键 |
| module | String | 模块名 | 必填 |
| field_name | String | 字段名 | 必填 |
| field_type | String | 字段类型 | 必填 (string/number/boolean/date/array/object) |
| description | String | 字段描述 | 可选 |
| required | Boolean | 是否必填 | 默认false |
| default_value | Any | 默认值 | 可选 |
| constraints | Object | 字段约束 | 可选 |
| created_by | String | 创建者 | 必填 |
| created_at | DateTime | 创建时间 | 自动生成 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 自动生成 |

### 7.4 字段约束说明 (constraints)

| 约束名 | 适用类型 | 描述 |
|--------|----------|------|
| type | string | 约束类型 |
| min | number | 最小值 |
| max | number | 最大值 |
| min_length | string | 最小长度 |
| max_length | string | 最大长度 |
| pattern | string | 正则表达式 |
| enum_values | string | 枚举值列表 |
| list_min_length | array | 数组最小长度 |
| list_max_length | array | 数组最大长度 |

### 7.5 业务数据表 ({module}_data)

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

### 7.6 已删除业务数据表 (deleted_data)

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
| created_at | DateTime | 创建时间 | 原始数据创建时间 |
| updated_by | String | 更新者 | 必填 |
| updated_at | DateTime | 更新时间 | 删除操作更新时间 |

### 7.7 已删除刮削任务表 (deleted_scrape_tasks)

| 字段名 | 数据类型 | 描述 | 约束 |
|--------|----------|------|------|
| _id | ObjectID | 删除记录ID | 主键 |
| module | String | 模块名 | 必填 |
| original_id | ObjectID | 原始任务ID | 必填 |
| data_path | String | 数据文件路径 | 必填 |
| scraper_path | String | 刮削器路径 | 必填 |
| status | String | 刮削状态 | pending/scraping/success/failed |
| result | Object | 刮削结果 | 可选 |
| error_message | String | 错误信息 | 可选 |
| started_at | DateTime | 开始刮削时间 | 可选 |
| completed_at | DateTime | 完成刮削时间 | 可选 |
| business_data_id | ObjectID | 关联的业务数据ID | 可选 |
| deleted_at | DateTime | 删除时间 | 自动生成 |
| created_at | DateTime | 创建时间 | 原始任务创建时间 |
| updated_at | DateTime | 更新时间 | 删除操作更新时间 |

## 8. 索引策略

### 8.1 集合索引

为提高查询性能，系统自动为各集合创建以下索引：

| 集合 | 索引字段 | 类型 | 描述 |
|------|----------|------|------|
| collections | module | 唯一索引 | 快速查找模块 |
| field_definitions | module, field_name | 复合唯一索引 | 按模块和字段名查询 |
| scrape_tasks | module, status | 复合索引 | 按模块和状态查询任务 |
| scrape_tasks | created_at | 降序索引 | 按时间排序查询 |

### 8.2 动态索引

由于MongoDB支持动态模式，刮削器的结果可能包含各种不同的字段，系统支持：

- **自动索引建议**：基于查询模式自动建议索引
- **手动索引管理**：可通过接口为特定字段创建索引
- **复合索引**：支持创建多字段复合索引提高复杂查询性能

### 8.3 索引管理API

- **POST /api/collections/:module/indexes**：创建索引
- **GET /api/collections/:module/indexes**：查看索引列表
- **DELETE /api/collections/:module/indexes/:name**：删除索引

## 9. 日志管理

### 9.1 日志类型

| 日志类型 | 描述 | 存储位置 |
|----------|------|----------|
| HTTP日志 | API请求响应记录 | logs/http.log |
| 应用日志 | 程序运行日志 | logs/app.log |

### 9.2 日志内容

#### HTTP日志

```json
{
  "level": "info",
  "time": "2024-01-15T10:00:00Z",
  "method": "POST",
  "path": "/api/scraper/upload",
  "status": 200,
  "latency": "45ms",
  "client_ip": "192.168.1.1"
}
```

#### 应用日志

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "caller": "scraper.go:123",
  "message": "刮削任务处理完成",
  "task_id": "task123",
  "module": "movie"
}
```

### 9.3 日志记录点

系统在以下关键节点记录日志：

1. **刮削相关**
   - 任务提交
   - 任务开始执行
   - 任务执行完成（成功/失败）
   - 刮削结果存储

2. **集合管理**
   - 集合创建
   - 集合更新
   - 集合删除
   - 索引管理操作

3. **数据操作**
   - 数据创建
   - 数据更新
   - 数据删除
   - 数据恢复

## 10. 基础结构要求

### 10.1 主键设计

- 业务数据使用MongoDB自带的_id作为主键(PK)

### 10.2 审计字段

所有业务数据需包含审计(audit)字段，记录：
- 创建者(created_by)
- 创建时间(created_at)
- 更新者(updated_by)
- 更新时间(updated_at)

### 10.3 datatypeowner职责

- datatypeowner拥有集合的管理权限
- datatypeowner可以更新集合信息
- datatypeowner身份可以转移

### 10.4 数据存储特点

- 利用MongoDB的JSON格式存储，支持无限多种数据结构
- 刮削器结果直接以JSON形式存储在custom_fields字段中
- 支持嵌套结构、数组等复杂数据类型
- 无需预先定义所有字段结构，适应不同刮削器的输出

### 10.5 并发处理

- 刮削任务采用异步并发处理
- 使用工作队列管理刮削任务
- 支持配置刮削任务的并发度（默认4个工作协程）
- 任务队列大小默认1000

# 业务数据模块 - 需求设计文档

## 1. 需求背景

数据中心的核心功能是存储和管理各种业务数据。数据按模块（如电影、音乐、书籍）分类，每个模块的数据结构可能不同（动态字段）。需要提供：

- 灵活的字段定义和验证
- 按模块组织的数据 CRUD 操作
- 高级查询能力（JQL）
- 数据安全删除与恢复（软删除）

## 2. 功能需求

### FR-BD-01: 字段定义管理

管理员可以为每个模块定义自定义字段及验证规则。

**输入**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 是 | 所属模块 |
| field_name | string | 是 | 字段名 |
| field_type | string | 是 | 字段类型 (string/number/boolean/date/array/object) |
| description | string | 否 | 字段描述 |
| required | bool | 否 | 是否必填 |
| constraints | object | 否 | 验证约束 |

**支持的字段类型与约束**:
| 类型 | 可用约束 |
|------|----------|
| string | min_length, max_length, pattern, enum_values |
| number | min, max |
| boolean | - |
| date | RFC3339 格式 |
| array | list_min_length, list_max_length |
| object | - |

### FR-BD-02: 业务数据 CRUD

| 操作 | 权限 | 说明 |
|------|------|------|
| 创建 | 集合 write | 必填字段验证通过后插入 |
| 查询列表 | 集合 read | 分页，支持 JQL 过滤 |
| 查询详情 | 集合 read | 按 ID 查找 |
| 更新 | 集合 write | 更新后验证字段 |
| 删除 | 集合 delete | 软删除到 deleted_data |

### FR-BD-03: 软删除与恢复

数据删除时不物理删除，而是移动到 `deleted_data` 集合，保留恢复能力。

**软删除记录**:
- `original_id`: 原始数据 ID
- `deleted_at`: 删除时间
- 保留完整的自定义字段数据

**恢复操作**: 将数据从 `deleted_data` 移回原模块集合。

### FR-BD-04: JQL 高级查询

支持在列表查询中使用 JQL 表达式进行复杂过滤。

```
GET /api/business/module/movie?jql=year>2000 AND rating>7.5
```

**支持的查询能力**:
- 比较运算符: =, !=, >, <, >=, <=, ~ (正则)
- 逻辑运算: AND, OR, NOT
- 列表匹配: IN, NOT IN
- 空值检查: IS NULL, IS NOT NULL
- 括号分组: (A AND B) OR C
- 时间函数: Now(), StartOfWeek(), EndOfMonth() 等

---

## 3. 数据存储设计

### 3.1 动态集合

每个模块的数据存储在独立的 MongoDB 集合中，命名规则为 `{module}_data`。

```
movie 模块 → movie_data 集合
music 模块 → music_data 集合
book  模块 → book_data 集合
```

### 3.2 字段存储策略

**系统字段**（文档根级别）:
```
_id, module, description, file_path,
created_by, created_at, updated_by, updated_at
```

**用户自定义字段**（custom_fields 嵌套文档）:
```
custom_fields: {
  title: "Harry Potter",
  year: 2001,
  rating: 7.6,
  ...
}
```

**设计原因**: 将用户字段嵌套在 `custom_fields` 下可以避免与系统字段冲突，同时保持文档结构清晰。

### 3.3 软删除存储

```
deleted_data 集合
{
  _id, module, original_id, description,
  custom_fields, file_path,
  deleted_at (新增),
  created_by, created_at, updated_by, updated_at
}
```

索引: `{module: 1}`, `{original_id: 1}`, `{deleted_at: 1}`

---

## 4. 非功能需求

| 编号 | 需求 | 说明 |
|------|------|------|
| NFR-BD-01 | 字段验证性能 | 单次验证在微秒级完成 |
| NFR-BD-02 | 数据隔离 | 不同模块数据存储在不同集合 |
| NFR-BD-03 | 软删除可追溯 | 保留删除时间、原始 ID |
| NFR-BD-04 | 写入审计 | 记录 created_by 和 updated_by |

---

## 5. 业务流程

### 5.1 创建业务数据

```
POST /api/business { module, data, description }
      │
      ▼
JWT 认证 + 集合写权限检查
      │
      ▼
模块集合是否存在？
  ├── 否 → 400 "请先创建模块集合"
  └── 是 ↓
获取模块字段定义
      │
      ▼
遍历字段定义进行验证
  ├── title → Validate(value) → 检查 required/type/constraints
  ├── year  → Validate(value) → 检查 type/min/max
  └── 任一失败 → 400 + { field, message }
      │
      ▼
构造 BusinessData → 插入 {module}_data
      │
      ▼
201 Created
```

### 5.2 删除与恢复

```
DELETE /api/business/module/movie/:id
      │
      ▼
1. 从 movie_data 查询数据
2. 构造 DeletedData { original_id, deleted_at, ... }
3. 插入 deleted_data
4. 从 movie_data 删除
      │
      ▼
200 OK

POST /api/deleted/:id/recover
      │
      ▼
1. 从 deleted_data 查询
2. 取出 module 和 original_id
3. 构造 BusinessData 插回 {module}_data
4. 从 deleted_data 删除
      │
      ▼
200 OK
```

## 6. 接口定义

### 创建数据

```json
POST /api/business
Request:
{
  "module": "movie",
  "description": "导入的电影数据",
  "data": {
    "title": "Harry Potter",
    "year": 2001,
    "rating": 7.6
  }
}
Response 200:
{
  "message": "数据创建成功",
  "data": { "_id": "...", "module": "movie", ... },
  "module": "movie"
}
```

### 查询数据

```json
GET /api/business/module/movie?page=1&pageSize=10&jql=year>2000
Response 200:
{
  "data": [...],
  "total": 50,
  "page": 1,
  "pageSize": 10
}
```

### 字段验证失败

```json
Response 400:
{
  "error": "字段验证失败",
  "errors": [
    { "field": "title", "message": "此字段为必填项" },
    { "field": "year", "message": "值必须大于等于 1900" }
  ],
  "module": "movie",
  "field": "title"
}
```

# 业务数据模块 - 技术文档

## 1. 模块概述

业务数据模块负责管理各业务模块（集合）下的数据记录，支持动态字段、字段验证、软删除与恢复、JQL 高级查询等功能。数据按模块存储在不同的 MongoDB 动态集合中。

### 模块位置

```
internal/api/handlers.go              # 业务数据 CRUD handler
internal/models/models.go             # BusinessData, DeletedData, FieldDefinition
internal/storage/mongodb_storage.go   # 业务数据存储层
pkg/jql/parser.go                     # JQL 查询引擎
```

---

## 2. 数据模型

### 2.1 BusinessData

```go
type BusinessData struct {
    ID           primitive.ObjectID     `json:"_id" bson:"_id"`
    Module       string                 `json:"module" bson:"module"`
    Description  string                 `json:"description" bson:"description"`
    CustomFields map[string]interface{} `json:"custom_fields" bson:"custom_fields"`
    FilePath     string                 `json:"file_path" bson:"file_path"`
    BaseModel
}
```

**存储位置**: 动态集合 `{module}_data`（如 `movie_data`）

**关键设计**:
- `CustomFields` 是 `map[string]interface{}`，实现真正的动态 Schema
- 系统字段（`_id`, `module`, `description`, `created_at` 等）在文档根级别
- 用户自定义字段嵌套在 `custom_fields` 下

### 2.2 DeletedData

```go
type DeletedData struct {
    ID           primitive.ObjectID     `json:"_id" bson:"_id"`
    OriginalID   primitive.ObjectID     `json:"original_id" bson:"original_id"`
    Module       string                 `json:"module" bson:"module"`
    Description  string                 `json:"description" bson:"description"`
    CustomFields map[string]interface{} `json:"custom_fields" bson:"custom_fields"`
    FilePath     string                 `json:"file_path" bson:"file_path"`
    DeletedAt    time.Time              `json:"deleted_at" bson:"deleted_at"`
    BaseModel
}
```

**存储位置**: `deleted_data` 集合

**索引**:
- `{module: 1}` — 按模块查询
- `{original_id: 1}` — 按原始 ID 定位
- `{deleted_at: 1}` — 按删除时间排序

### 2.3 FieldDefinition

```go
type FieldDefinition struct {
    ID           primitive.ObjectID `json:"_id" bson:"_id"`
    Module       string             `json:"module" bson:"module"`
    FieldName    string             `json:"field_name" bson:"field_name"`
    FieldType    FieldType          `json:"field_type" bson:"field_type"`
    Description  string             `json:"description" bson:"description"`
    Required     bool               `json:"required" bson:"required"`
    DefaultValue interface{}        `json:"default_value,omitempty" bson:"default_value,omitempty"`
    Constraints  Constraints        `json:"constraints" bson:"constraints"`
    BaseModel
}
```

### 2.4 字段类型与约束

```go
type FieldType string
const (
    FieldTypeString  FieldType = "string"
    FieldTypeNumber  FieldType = "number"
    FieldTypeBoolean FieldType = "boolean"
    FieldTypeDate    FieldType = "date"
    FieldTypeArray   FieldType = "array"
    FieldTypeObject  FieldType = "object"
)

type Constraints struct {
    Type       ConstraintType `json:"type"`
    Min        *float64       `json:"min,omitempty"`        // number
    Max        *float64       `json:"max,omitempty"`        // number
    MinLength  *int           `json:"min_length,omitempty"` // string
    MaxLength  *int           `json:"max_length,omitempty"` // string
    Pattern    string         `json:"pattern,omitempty"`    // string regex
    EnumValues []string       `json:"enum_values,omitempty"` // string
    ListMinLen *int           `json:"list_min_length,omitempty"` // array
    ListMaxLen *int           `json:"list_max_length,omitempty"` // array
}
```

---

## 3. 字段验证

### 3.1 Validate 方法

```go
func (f *FieldDefinition) Validate(value interface{}) *FieldValidationResult
```

验证结果结构：
```go
type FieldValidationResult struct {
    Valid  bool
    Errors []FieldValidationError
}
type FieldValidationError struct {
    Field   string
    Message string
}
```

### 3.2 验证规则矩阵

| 字段类型 | 验证规则 |
|----------|----------|
| **number** | 类型检查(float64/int)、Min、Max |
| **string** | 类型检查(string)、MinLength、MaxLength、Pattern(正则)、EnumValues |
| **boolean** | 类型检查(bool) |
| **date** | 类型检查(string)+ RFC3339 格式校验 |
| **array** | 类型检查([]interface{})、ListMinLen、ListMaxLen |
| **object** | 基本非空校验 |

### 3.3 在创建/更新中触发验证

```go
// 创建和更新时验证
fieldDefs, _ := h.storage.GetFieldDefinitionsByModule(req.Module)
for _, fieldDef := range fieldDefs {
    value := req.Data[fieldDef.FieldName]
    result := fieldDef.Validate(value)
    if !result.Valid {
        return 400 { error: "字段验证失败", errors: result.Errors }
    }
}
```

---

## 4. 存储层

### 4.1 创建业务数据

```go
CreateBusinessData(ctx context.Context, collectionName string, data *models.BusinessData) error
```

将 BusinessData 插入动态集合 `{module}_data`。

### 4.2 查询业务数据

```go
GetBusinessDataByModule(module string, filter bson.M, skip, limit int64) ([]models.BusinessData, error)
GetBusinessDataByID(module string, id string) (*models.BusinessData, error)
GetBusinessDataCount(module string, filter bson.M) (int64, error)
```

支持 MongoDB filter + 分页。

### 4.3 更新业务数据

```go
UpdateBusinessData(data *models.BusinessData) error
```

更新时需要验证字段，并记录 UpdatedBy 和 UpdatedAt。

### 4.4 删除业务数据（软删除）

```go
DeleteBusinessData(id string, deletedBy string) error
```

**流程**:
1. 从原始集合找出数据
2. 构造 `DeletedData`（含 original_id 和 deleted_at）
3. 插入 `deleted_data` 集合
4. 从原始集合删除

### 4.5 恢复业务数据

```go
RecoverDeletedData(id string, recoveredBy string) error
```

**流程**:
1. 从 `deleted_data` 根据 id 找出记录
2. 取出 `original_id`
3. 根据 `module` 字段构造 BusinessData，插入回 `{module}_data`
4. 从 `deleted_data` 删除

---

## 5. JQL 查询集成

业务数据列表查询支持 JQL 查询参数：

```
GET /api/business/module/movie?jql=status="active" AND year>2000
```

### 5.1 JQL → MongoDB 过滤

```go
jqlQuery := c.Query("jql")
filter := bson.M{}
if jqlQuery != "" {
    filter, _ = jql.ParseQuery(jqlQuery)
    filter = prefixCustomFields(filter) // 添加 custom_fields. 前缀
}
```

### 5.2 字段名前缀转换

`prefixCustomFields()` 函数将 JQL 中的用户字段名转换为 MongoDB 嵌套路径：

```
JQL:  title = "Harry Potter"
  ↓
MongoDB:  { "custom_fields.title": "Harry Potter" }
```

**系统字段名**（不添加前缀）:
```
_id, module, description, created_at, updated_at,
created_by, updated_by, data_path, file_path, custom_fields
```

---

## 6. API 接口

### 6.1 业务数据 CRUD

| 方法 | 路径 | 所需权限 | 说明 |
|------|------|----------|------|
| POST | /api/business | 集合 write | 创建数据 |
| GET | /api/business/module/:module | 集合 read | 查询（支持 JQL） |
| GET | /api/business/module/:module/:id | 集合 read | 详情 |
| PUT | /api/business/module/:module/:id | 集合 write | 更新 |
| DELETE | /api/business/module/:module/:id | 集合 delete | 软删除 |

### 6.2 已删除数据

| 方法 | 路径 | 所需权限 | 说明 |
|------|------|----------|------|
| GET | /api/deleted/module/:module | data:read | 按模块查看已删除数据 |
| GET | /api/deleted/:id | data:read | 查看已删除数据详情 |
| POST | /api/deleted/:id/recover | data:write | 恢复数据 |

---

## 7. 创建业务数据完整流程

```
POST /api/business
{
  "module": "movie",
  "description": "导入的电影数据",
  "data": { "title": "Harry Potter", "year": 2001 }
}
      │
      ▼
1. 权限检查: CollectionPermissionMiddleware(":write") → 检查 movie:write
      │
      ▼
2. 检查模块集合是否存在（GetCollectionByModule）
      │
      ▼
3. 获取字段定义 (GetFieldDefinitionsByModule)
      │
      ▼
4. 遍历字段定义进行验证
   ├── FieldDefinition.Validate(value)
   └── 验证失败 → 返回 400 + 验证错误详情
      │
      ▼
5. 构造 BusinessData
   {
     module: "movie",
     description: "导入的电影数据",
     custom_fields: { title: "Harry Potter", year: 2001 },
     created_by: userID, created_at: now,
     updated_by: userID, updated_at: now
   }
      │
      ▼
6. 插入 movie_data 集合
      │
      ▼
7. 返回 200 { message, data, module }
```

# 业务数据模块 - 需求实现文档

## 1. 实现概述

业务数据模块的实现分布在以下层面：
- **Handler**: HTTP 请求处理 + 字段验证触发 + JQL 集成
- **Models**: 字段定义的 Validate 方法
- **Storage**: MongoDB 动态集合 CRUD + 软删除

---

## 2. 文件清单

| 文件 | 说明 |
|------|------|
| `internal/api/handlers.go` | CreateBusinessData, GetBusinessDataByModule, UpdateBusinessData, DeleteBusinessData, GetDeletedData*, RecoverDeletedData |
| `internal/models/models.go` | BusinessData, DeletedData, FieldDefinition, Constraints, Validate() |
| `internal/storage/mongodb_storage.go` | 业务数据存储 CRUD + 软删除 |
| `pkg/jql/parser.go` | JQL → MongoDB filter 转换 |

---

## 3. 创建业务数据实现

```go
// internal/api/handlers.go:942-1011
func (h *Handler) CreateBusinessData(c *gin.Context) {
    var req struct {
        Module       string                 `json:"module" binding:"required"`
        Data         map[string]interface{} `json:"data"`
        Description  string                 `json:"description"`
        CustomFields map[string]interface{} `json:"custom_fields"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 1. 检查模块集合是否存在
    collection, err := h.storage.GetCollectionByModule(req.Module)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": fmt.Sprintf("模块不存在: %s，请先创建模块集合", req.Module),
        })
        return
    }

    // 2. 获取字段定义并验证
    fieldDefs, err := h.storage.GetFieldDefinitionsByModule(req.Module)
    if err == nil && len(fieldDefs) > 0 {
        for _, fieldDef := range fieldDefs {
            value := req.Data[fieldDef.FieldName]
            result := fieldDef.Validate(value)
            if !result.Valid {
                c.JSON(http.StatusBadRequest, gin.H{
                    "error":  "字段验证失败",
                    "errors": result.Errors,
                    "module": req.Module,
                    "field":  fieldDef.FieldName,
                })
                return
            }
        }
    }

    // 3. 构造 BusinessData
    userIDStr := "unknown"
    if userID, exists := c.Get("user_id"); exists {
        userIDStr = userID.(string)
    }

    data := &models.BusinessData{
        Module:      req.Module,
        Description: req.Description,
        BaseModel: models.BaseModel{
            CreatedBy: userIDStr, CreatedAt: time.Now(),
            UpdatedBy: userIDStr, UpdatedAt: time.Now(),
        },
    }
    if req.Data != nil {
        data.CustomFields = req.Data
    }
    if req.CustomFields != nil {
        data.CustomFields = req.CustomFields
    }

    // 4. 插入数据库
    ctx := context.Background()
    if err := h.storage.CreateBusinessData(ctx, collection.CollectionName, data); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "数据创建成功",
        "data":    data,
        "module":  req.Module,
    })
}
```

**关键点**:
- 支持 `data` 和 `custom_fields` 两种入参方式，`data` 优先
- 验证失败返回详细的字段级错误信息
- 自动填充审计字段（created_by/updated_by）

---

## 4. JQL 查询实现

```go
// internal/api/handlers.go:1013-1049
func (h *Handler) GetBusinessDataByModule(c *gin.Context) {
    module := c.Param("module")
    page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
    pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
    skip := (page - 1) * pageSize

    // JQL 查询解析
    jqlQuery := c.Query("jql")
    filter := bson.M{}
    if jqlQuery != "" {
        var err error
        filter, err = jql.ParseQuery(jqlQuery)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Invalid JQL query: " + err.Error(),
            })
            return
        }
        // 关键步骤：为字段名添加 custom_fields. 前缀
        filter = prefixCustomFields(filter)
    }

    dataList, _ := h.storage.GetBusinessDataByModule(module, filter, skip, pageSize)
    total, _ := h.storage.GetBusinessDataCount(module, filter)

    c.JSON(http.StatusOK, gin.H{
        "data":     dataList,
        "total":    total,
        "page":     page,
        "pageSize": pageSize,
    })
}
```

---

## 5. prefixCustomFields 转换函数

```go
// internal/api/handlers.go:1775-1833
var systemFieldNames = map[string]bool{
    "_id": true, "module": true, "description": true,
    "created_at": true, "updated_at": true,
    "created_by": true, "updated_by": true,
    "data_path": true, "file_path": true, "custom_fields": true,
}

func prefixCustomFields(m bson.M) bson.M {
    result := bson.M{}
    for k, v := range m {
        switch k {
        case "$and", "$or":
            // 递归处理子条件数组
            arr, ok := v.([]bson.M)
            if ok {
                prefixed := make([]bson.M, len(arr))
                for i, item := range arr {
                    prefixed[i] = prefixCustomFields(item)
                }
                result[k] = prefixed
            } else if arr2, ok := v.([]interface{}); ok {
                prefixed := make([]bson.M, len(arr2))
                for i, item := range arr2 {
                    if bsm, ok := item.(bson.M); ok {
                        prefixed[i] = prefixCustomFields(bsm)
                    }
                }
                result[k] = prefixed
            }
        case "$not":
            if vm, ok := v.(bson.M); ok {
                result[k] = prefixCustomFields(vm)
            }
        default:
            // 非系统字段 → 添加 custom_fields. 前缀
            if vv, ok := v.(bson.M); ok {
                hasOpKey := false
                for subKey := range vv {
                    if len(subKey) > 0 && subKey[0] == '$' {
                        hasOpKey = true
                        break
                    }
                }
                if hasOpKey {
                    if !systemFieldNames[k] && k[0] != '$' {
                        result["custom_fields."+k] = vv
                    } else {
                        result[k] = vv
                    }
                } else {
                    result[k] = prefixCustomFields(vv)
                }
            } else if !systemFieldNames[k] && k[0] != '$' {
                result["custom_fields."+k] = v
            } else {
                result[k] = v
            }
        }
    }
    return result
}
```

**转换示例**:
```
JQL: title = "HP" AND year > 2000
  ↓ ParseQuery
{ "$and": [{ "title": "HP" }, { "year": { "$gt": 2000 } }] }
  ↓ prefixCustomFields
{ "$and": [
    { "custom_fields.title": "HP" },
    { "custom_fields.year": { "$gt": 2000 } }
] }
```

---

## 6. 字段验证实现

```go
// internal/models/models.go:73-221
func (f *FieldDefinition) Validate(value interface{}) *FieldValidationResult {
    result := &FieldValidationResult{Valid: true, Errors: []FieldValidationError{}}

    // Required 检查
    if f.Required && (value == nil || value == "") {
        result.Valid = false
        result.Errors = append(result.Errors, FieldValidationError{
            Field: f.FieldName, Message: "此字段为必填项",
        })
        return result
    }
    if value == nil || value == "" {
        return result
    }

    // 按类型验证
    switch f.FieldType {
    case FieldTypeNumber:
        numVal, ok := value.(float64)
        if !ok {
            // 尝试 int 转换
            intVal, ok := value.(int)
            if !ok {
                result.Valid = false
                result.Errors = append(...)
                return result
            }
            numVal = float64(intVal)
        }
        if f.Constraints.Min != nil && numVal < *f.Constraints.Min {
            result.Errors = append(...) // "值必须大于等于 X"
        }
        if f.Constraints.Max != nil && numVal > *f.Constraints.Max {
            result.Errors = append(...) // "值必须小于等于 X"
        }

    case FieldTypeString:
        strVal, ok := value.(string)
        if !ok { /* 类型错误 */ }
        if f.Constraints.MinLength != nil && len(strVal) < *f.Constraints.MinLength { /* 长度不足 */ }
        if f.Constraints.MaxLength != nil && len(strVal) > *f.Constraints.MaxLength { /* 长度超限 */ }
        if f.Constraints.Pattern != "" {
            matched, _ := regexp.MatchString(f.Constraints.Pattern, strVal)
            if !matched { /* 正则不匹配 */ }
        }
        if len(f.Constraints.EnumValues) > 0 {
            found := false
            for _, enumVal := range f.Constraints.EnumValues {
                if strVal == enumVal { found = true; break }
            }
            if !found { /* 不在枚举值中 */ }
        }

    case FieldTypeBoolean:
        _, ok := value.(bool)
        if !ok { /* 类型错误 */ }

    case FieldTypeDate:
        if strVal, ok := value.(string); ok {
            _, err := time.Parse(time.RFC3339, strVal)
            if err != nil { /* 格式错误 */ }
        }

    case FieldTypeArray:
        arrVal, ok := value.([]interface{})
        if !ok { /* 类型错误 */ }
        if f.Constraints.ListMinLen != nil && len(arrVal) < *f.Constraints.ListMinLen { /* 长度不足 */ }
        if f.Constraints.ListMaxLen != nil && len(arrVal) > *f.Constraints.ListMaxLen { /* 长度超限 */ }
    }

    return result
}
```

---

## 7. 软删除与恢复实现

### 7.1 删除

```go
// internal/api/handlers.go:1127-1141
func (h *Handler) DeleteBusinessData(c *gin.Context) {
    id := c.Param("id")
    userIDStr := "unknown"
    if userID, exists := c.Get("user_id"); exists {
        userIDStr = userID.(string)
    }
    if err := h.storage.DeleteBusinessData(id, userIDStr); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Business data deleted successfully"})
}
```

**存储层实现**（`mongodb_storage.go`）:
1. 从动态集合查找原始数据
2. 构造 `DeletedData`（复制所有字段 + original_id + deleted_at）
3. 插入 `deleted_data` 集合
4. 从动态集合删除原始文档

### 7.2 恢复

```go
// internal/api/handlers.go:1173-1187
func (h *Handler) RecoverDeletedData(c *gin.Context) {
    id := c.Param("id")
    userIDStr := "unknown"
    if userID, exists := c.Get("user_id"); exists {
        userIDStr = userID.(string)
    }
    if err := h.storage.RecoverDeletedData(id, userIDStr); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Data recovered successfully"})
}
```

**存储层实现**:
1. 从 `deleted_data` 查找记录
2. 根据 `module` 确定集合名（`{module}_data`）
3. 构造 `BusinessData` 并插入回动态集合
4. 从 `deleted_data` 删除记录

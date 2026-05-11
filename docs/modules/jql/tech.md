# JQL 查询模块 - 技术文档

## 1. 模块概述

JQL (JSON Query Language) 是数据中心自定义的查询表达式语言，通过手写递归下降解析器将类 SQL 的查询语法转换为 MongoDB `bson.M` 过滤器。模块完全自包含在 `pkg/jql/` 中，无外部依赖。

### 模块位置

```
pkg/jql/
├── parser.go           # 词法分析 + 语法分析 + MongoDB 转换
└── parser_test.go      # 单元测试
```

### 对外 API

```go
func ParseQuery(query string) (bson.M, error)       // 解析 JQL → MongoDB filter
func ValidateJQL(query string) error                 // 验证 JQL 语法
func GetExampleQueries() []string                    // 获取示例查询
```

---

## 2. 词法分析 (Tokenizer)

### 2.1 Token 类型

```go
type TokenType string

const (
    TokenTypeField      TokenType = "field"       // 字段名
    TokenTypeOperator   TokenType = "operator"    // =, !=, >, <, >=, <=, ~
    TokenTypeValue      TokenType = "value"       // 字面量值
    TokenTypeFunction   TokenType = "function"    // Now(), StartOfWeek() 等
    TokenTypeLeftParen  TokenType = "left_paren"  // (
    TokenTypeRightParen TokenType = "right_paren" // )
    TokenTypeAnd        TokenType = "and"         // AND
    TokenTypeOr         TokenType = "or"          // OR
    TokenTypeNot        TokenType = "not"         // NOT
    TokenTypeComma      TokenType = "comma"       // ,
    TokenTypeIn         TokenType = "in"          // IN
    TokenTypeNotIn      TokenType = "not_in"      // NOT IN
    TokenTypeLike       TokenType = "like"        // ~ (正则)
    TokenTypeIsNull     TokenType = "is_null"     // IS NULL
    TokenTypeIsNotNull  TokenType = "is_not_null" // IS NOT NULL
)
```

### 2.2 词法分析流程

`tokenize(query string) ([]Token, error)` 逐字符扫描：

```
扫描优先级（从高到低）:
1. 关键字: AND, OR, NOT, NOT IN, IN, IS NULL, IS NOT NULL
2. 括号和逗号: (, ), ,
3. 操作符: >=, <=, !=, =, >, <, ~
4. 数字值: -?\d+(\.\d+)?
5. 函数名: CurrentUser, Now, StartOfDay, EndOfDay, StartOfWeek, EndOfWeek, StartOfMonth, EndOfMonth
6. 字段名: [a-zA-Z0-9_\.]+
7. 字符串值: '...' 或 "..."
```

**关键字识别规则**: 关键字后必须跟非字母数字字符，防止误匹配字段名（如 `name` 不会匹配 `name` 中的片段）。

---

## 3. 语法分析 (Parser)

### 3.1 递归下降解析器

```go
type Parser struct {
    tokens []Token
    pos    int
}
```

### 3.2 语法层次

```
Expression      → OrExpression
OrExpression    → AndExpression ("OR" AndExpression)*
AndExpression   → NotExpression ("AND" NotExpression)*
NotExpression   → "NOT"? PrimaryExpression
PrimaryExpression → "(" Expression ")"
                  | Condition

Condition       → Field Operator Value
                | Field "IN" "(" ValueList ")"
                | Field "NOT IN" "(" ValueList ")"
                | Field "IS NULL"
                | Field "IS NOT NULL"

ValueList       → Value ("," Value)*

Operator        → "=" | "!=" | ">" | "<" | ">=" | "<=" | "~"

Value           → TokenTypeValue | TokenTypeFunction
```

### 3.3 递归下降实现

```
parseExpression()
  └── parseOrExpression()        // 处理 OR
        └── parseAndExpression()   // 处理 AND
              └── parseNotExpression()  // 处理 NOT
                    └── parsePrimaryExpression()  // 处理括号和条件
                          ├── parseExpression()       // 括号递归
                          └── parseCondition()         // 字段 + 操作符 + 值
                                ├── parseValue()        // 字面量
                                └── parseFunction()     // 函数求值
```

**AST 中间表示**: `interface{}`（可能是 `bson.M`、`map[string]interface{}`、`[]interface{}`）

---

## 4. MongoDB 转换

### 4.1 运算符映射

```go
func (p *Parser) convertCondition(field, operator string, value interface{}) bson.M {
    switch operator {
    case "=":       return bson.M{field: value}
    case "!=":      return bson.M{field: bson.M{"$ne": value}}
    case ">":       return bson.M{field: bson.M{"$gt": value}}
    case "<":       return bson.M{field: bson.M{"$lt": value}}
    case ">=":      return bson.M{field: bson.M{"$gte": value}}
    case "<=":      return bson.M{field: bson.M{"$lte": value}}
    case "~":       return bson.M{field: bson.M{"$regex": value, "$options": "i"}}
    case "IN":      return bson.M{field: bson.M{"$in": value}}
    case "NOT IN":  return bson.M{field: bson.M{"$nin": value}}
    case "IS NULL":     return bson.M{field: bson.M{"$exists": false}}
    case "IS NOT NULL": return bson.M{field: bson.M{"$exists": true}}
    }
}
```

**`=` 运算符特殊处理**: 当 value 为 nil 时转为 `$exists: false`（与 IS NULL 等价）。

### 4.2 逻辑运算映射

| JQL | AST | MongoDB |
|-----|-----|---------|
| `A AND B` | `{"$and": [A, B]}` | `{"$and": [...]}` |
| `A OR B` | `{"$or": [A, B]}` | `{"$or": [...]}` |
| `NOT A` | `{"$not": A}` | `{"$not": ...}` |

### 4.3 convertToMongoQuery

递归将 AST 转换为纯 `bson.M`：

```go
func (p *Parser) convertToMongoQuery(ast interface{}) bson.M {
    switch v := ast.(type) {
    case bson.M:
        // 处理 $and, $or, $not 递归
    case map[string]interface{}:
        // 同上
    default:
        return bson.M{}
    }
}
```

---

## 5. 内置函数

### 5.1 函数列表

| 函数 | 返回值 | 说明 |
|------|--------|------|
| `Now()` | `time.Now()` | 当前时间 |
| `StartOfDay()` | `today 00:00:00` | 当天开始 |
| `EndOfDay()` | `today 23:59:59.999...` | 当天结束 |
| `StartOfWeek()` | `monday 00:00:00` | 本周一开始 |
| `EndOfWeek()` | `sunday 23:59:59.999...` | 本周日结束 |
| `StartOfMonth()` | `1st 00:00:00` | 本月开始 |
| `EndOfMonth()` | `last day 23:59:59.999...` | 本月结束 |
| `CurrentUser()` | `"currentUser()"` (string) | 当前用户（后续替换） |

### 5.2 函数名识别

```go
var knownFunctions = []string{
    "CurrentUser", "Now", "StartOfDay", "EndOfDay",
    "StartOfWeek", "EndOfWeek", "StartOfMonth", "EndOfMonth",
}

func isFunctionName(query string) bool {
    for _, fn := range knownFunctions {
        if strings.HasPrefix(strings.ToLower(query), strings.ToLower(fn)) {
            endIdx := len(fn)
            if endIdx < len(query) {
                // 函数名后跟 '(' 才识别为函数
                if query[endIdx] == '(' {
                    return true
                }
            }
        }
    }
    return false
}
```

---

## 6. 与业务数据集成

### 6.1 查询入口

```go
// internal/api/handlers.go:1019-1029
jqlQuery := c.Query("jql")
filter := bson.M{}
if jqlQuery != "" {
    filter, _ = jql.ParseQuery(jqlQuery)
    filter = prefixCustomFields(filter) // 添加 custom_fields. 前缀
}
```

### 6.2 prefixCustomFields 转换

由于业务数据的自定义字段存储在 `custom_fields` 嵌套文档中，JQL 解析后的字段名需要添加前缀。

**转换规则**:
- 系统字段名（`_id`, `module`, `description`, `created_at`, `updated_at`, `created_by`, `updated_by`, `data_path`, `file_path`, `custom_fields`）→ 不添加前缀
- MongoDB 操作符（以 `$` 开头）→ 不添加前缀
- 其他字段 → 添加 `custom_fields.` 前缀

**示例**:
```
JQL输入: title = "HP" AND year > 2000
  ↓ ParseQuery
{ "$and": [{ "title": "HP" }, { "year": { "$gt": 2000 } }] }
  ↓ prefixCustomFields
{ "$and": [
    { "custom_fields.title": "HP" },
    { "custom_fields.year": { "$gt": 2000 } }
] }
```

---

## 7. 查询示例

```go
func GetExampleQueries() []string {
    return []string{
        `status = "active"`,
        `name ~ "产品"`,
        `price > 100`,
        `status IN ("active", "pending")`,
        `category NOT IN ("deleted", "archived")`,
        `title ~ "重要"`,
        `created > "2024-01-01"`,
        `assignee IS NULL`,
        `email IS NOT NULL`,
        `status = "active" AND price > 100`,
        `name = "A" OR name = "B"`,
        `(status = "active") AND (price > 100 OR price < 50)`,
        `created > StartOfWeek() AND module = "movie"`,
        `updated < EndOfMonth() AND status NOT IN ("deleted", "archived")`,
    }
}
```

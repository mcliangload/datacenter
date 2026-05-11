# JQL 查询模块 - 需求实现文档

## 1. 实现概述

JQL 查询引擎是一个手写的递归下降解析器，全部实现在 `pkg/jql/parser.go` 中（约 650 行）。分为三个阶段：
1. **词法分析** (tokenize) → Token 流
2. **语法分析** (parseExpression et al.) → AST
3. **MongoDB 转换** (convertToMongoQuery) → `bson.M`

---

## 2. 文件清单

| 文件 | 说明 |
|------|------|
| `pkg/jql/parser.go` | 词法分析 + 语法分析 + MongoDB 转换 + 公开 API |
| `pkg/jql/parser_test.go` | 单元测试 |
| `internal/api/handlers.go` | HTTP handler 中使用 `jql.ParseQuery()` + `prefixCustomFields()` |

---

## 3. 公开 API

```go
// pkg/jql/parser.go:625-628
func ParseQuery(query string) (bson.M, error) {
    parser := NewParser()
    return parser.Parse(query)
}

// pkg/jql/parser.go:649-652
func ValidateJQL(query string) error {
    _, err := ParseQuery(query)
    return err
}

// pkg/jql/parser.go:630-647
func GetExampleQueries() []string {
    return []string{ /* 14 个示例 */ }
}
```

---

## 4. 词法分析实现

### 4.1 主扫描循环

```go
// pkg/jql/parser.go:67-230
func (p *Parser) tokenize(query string) ([]Token, error) {
    var tokens []Token
    query = strings.TrimSpace(query)

    for len(query) > 0 {
        query = strings.TrimSpace(query)
        lowerQuery := strings.ToLower(query)

        // 优先级从高到低扫描：
        
        // 1. 关键字: AND (3字符), OR (2), NOT (3), NOT IN (6),
        //             IN (2), IS NULL (7), IS NOT NULL (11)
        //    关键：关键字后必须是非字母数字字符
        if strings.HasPrefix(lowerQuery, "and") &&
           (len(query) == 3 || !isAlphanumeric(byte(query[3]))) {
            tokens = append(tokens, Token{Type: TokenTypeAnd, Value: "AND"})
            query = query[3:]
            continue
        }
        // ... 类似地处理其他关键字 ...

        // 2. 括号和逗号
        if strings.HasPrefix(query, "(") {
            tokens = append(tokens, Token{Type: TokenTypeLeftParen, Value: "("})
            query = query[1:]
            continue
        }
        // ...

        // 3. 操作符：>=, <=, !=, =, >, <, ~
        operators := []string{">=", "<=", "!=", "=", ">", "<", "~"}
        for _, op := range operators {
            if strings.HasPrefix(query, op) {
                tokens = append(tokens, Token{Type: TokenTypeOperator, Value: op})
                query = query[len(op):]
                found = true
                break
            }
        }

        // 4. 数字值（含正负号）
        if (query[0] == '-' || query[0] == '+') && ... { /* 有符号数字 */ }
        if query[0] >= '0' && query[0] <= '9' { /* 无符号数字 */ }

        // 5. 函数名
        if isFunctionName(query) { /* ... */ }

        // 6. 字段名
        if isFieldName(query[0]) { /* ... */ }

        // 7. 字符串值（单/双引号）
        if query[0] == '\'' || query[0] == '"' {
            quote := query[0]
            end := strings.Index(query[1:], string(quote))
            if end == -1 {
                return nil, errors.New("unclosed string")
            }
            value := query[1 : end+1]
            tokens = append(tokens, Token{Type: TokenTypeValue, Value: value})
            query = query[end+2:]
            continue
        }

        return nil, fmt.Errorf("unexpected character: %s", string(query[0]))
    }
    return tokens, nil
}
```

### 4.2 关键字识别规则

**关键设计**: 关键字后必须跟随非字母数字字符，防止误匹配。

```go
// 正确: "and price > 100" → AND + field(price) + > + value(100)
// 错误: "anderson" 不会匹配 AND, 因为 'e' 是字母

func isAlphanumeric(c byte) bool {
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
```

### 4.3 数字值识别

```go
// 有符号数字: -?\d+(\.\d+)?
if (query[0] == '-' || query[0] == '+') && len(query) > 1 &&
   query[1] >= '0' && query[1] <= '9' {
    end := 1
    hasDot := false
    for end < len(query) {
        c := query[end]
        if c == '.' {
            if hasDot { break }
            hasDot = true
            end++
            continue
        }
        if c < '0' || c > '9' { break }
        end++
    }
    tokens = append(tokens, Token{Type: TokenTypeValue, Value: query[:end]})
    query = query[end:]
    continue
}
```

### 4.4 字符串值识别

```go
if query[0] == '\'' || query[0] == '"' {
    quote := query[0]
    end := strings.Index(query[1:], string(quote))
    if end == -1 {
        return nil, errors.New("unclosed string")
    }
    value := query[1 : end+1]  // 不含引号的内容
    tokens = append(tokens, Token{Type: TokenTypeValue, Value: value})
    query = query[end+2:]  // 跳过闭合引号
    continue
}
```

---

## 5. 语法分析实现

### 5.1 Parser 结构

```go
type Parser struct {
    tokens []Token
    pos    int
}
```

### 5.2 运算符优先级

通过递归下降的调用层次自然表达：

```
parseExpression        (入口)
  └── parseOrExpression      (最低优先级: OR)
        └── parseAndExpression  (中优先级: AND)
              └── parseNotExpression (高优先级: NOT)
                    └── parsePrimaryExpression (最高: 括号/条件)
```

### 5.3 parseOrExpression 实现

```go
// pkg/jql/parser.go:272-288
func (p *Parser) parseOrExpression() (interface{}, error) {
    left, err := p.parseAndExpression()
    if err != nil {
        return nil, err
    }

    for p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenTypeOr {
        p.pos++
        right, err := p.parseAndExpression()
        if err != nil {
            return nil, err
        }
        left = map[string]interface{}{"$or": []interface{}{left, right}}
    }

    return left, nil
}
```

### 5.4 parsePrimaryExpression 实现

```go
// pkg/jql/parser.go:321-421
func (p *Parser) parsePrimaryExpression() (interface{}, error) {
    token := p.tokens[p.pos]

    // 处理括号: ( expr )
    if token.Type == TokenTypeLeftParen {
        p.pos++
        expr, err := p.parseExpression()
        if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenTypeRightParen {
            return nil, errors.New("expected closing parenthesis")
        }
        p.pos++
        return expr, nil
    }

    // 处理条件: field op value
    if token.Type == TokenTypeField {
        p.pos++
        fieldName := token.Value

        // IS NULL / IS NOT NULL
        if p.tokens[p.pos].Type == TokenTypeIsNull {
            p.pos++
            return p.convertCondition(fieldName, "IS NULL", nil), nil
        }
        if p.tokens[p.pos].Type == TokenTypeIsNotNull {
            p.pos++
            return p.convertCondition(fieldName, "IS NOT NULL", nil), nil
        }

        // IN / NOT IN
        opToken := p.tokens[p.pos]
        operator := opToken.Value
        p.pos++

        if operator == "IN" || operator == "NOT IN" {
            // 解析 (... ) 中的值列表
            var values []interface{}
            // ... 解析逗号分隔的值 ...
            return p.convertCondition(fieldName, operator, values), nil
        }

        // 普通操作符 (=, !=, >, <, etc.)
        valueToken := p.tokens[p.pos]
        var value interface{}
        if valueToken.Type == TokenTypeFunction {
            value = p.parseFunction(valueToken.Value)
        } else {
            value = p.parseValue()
        }
        return p.convertCondition(fieldName, operator, value), nil
    }

    return nil, fmt.Errorf("unexpected token: %s", token.Value)
}
```

### 5.5 值类型解析

```go
// pkg/jql/parser.go:441-463
func (p *Parser) parseValueType(value string) interface{} {
    if isInteger(value) {
        intVal := 0
        fmt.Sscanf(value, "%d", &intVal)
        return intVal
    }
    if isFloat(value) {
        var floatVal float64
        fmt.Sscanf(value, "%f", &floatVal)
        return floatVal
    }
    lower := strings.ToLower(value)
    if lower == "true"  { return true }
    if lower == "false" { return false }
    return value // 默认字符串
}
```

---

## 6. MongoDB 转换实现

### 6.1 运算符转换

```go
// pkg/jql/parser.go:538-574
func (p *Parser) convertCondition(field, operator string, value interface{}) bson.M {
    switch operator {
    case "=":
        if value == nil {
            return bson.M{field: bson.M{"$exists": false}}
        }
        return bson.M{field: value}
    case "!=":
        if value == nil {
            return bson.M{field: bson.M{"$exists": true}}
        }
        return bson.M{field: bson.M{"$ne": value}}
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

### 6.2 AST → bson.M 递归转换

```go
// pkg/jql/parser.go:576-623
func (p *Parser) convertToMongoQuery(ast interface{}) bson.M {
    switch v := ast.(type) {
    case bson.M:
        result := bson.M{}
        for key, val := range v {
            switch key {
            case "$and", "$or":
                var conditions []bson.M
                for _, item := range val.([]interface{}) {
                    conditions = append(conditions, p.convertToMongoQuery(item))
                }
                result[key] = conditions
            case "$not":
                result[key] = p.convertToMongoQuery(val)
            default:
                if vm, ok := val.(bson.M); ok {
                    result[key] = p.convertToMongoQuery(vm)
                } else {
                    result[key] = val
                }
            }
        }
        return result
    case map[string]interface{}:
        // 同样的处理逻辑（兼容 map）
    }
    return bson.M{}
}
```

---

## 7. 函数求值实现

```go
// pkg/jql/parser.go:502-536
func (p *Parser) parseFunction(funcName string) interface{} {
    switch strings.ToLower(funcName) {
    case "currentuser":
        return "currentUser()"
    case "now":
        return time.Now()
    case "startofday":
        return time.Now().Truncate(24 * time.Hour)
    case "endofday":
        now := time.Now()
        return time.Date(now.Year(), now.Month(), now.Day(),
            23, 59, 59, 999999999, now.Location())
    case "startofweek":
        now := time.Now()
        weekday := int(now.Weekday())
        if weekday == 0 { weekday = 7 } // 周日 → 7
        return now.AddDate(0, 0, -(weekday-1)).Truncate(24 * time.Hour)
    case "endofweek":
        now := time.Now()
        weekday := int(now.Weekday())
        if weekday == 0 { weekday = 7 }
        endOfWeek := now.AddDate(0, 0, 7-weekday)
        return time.Date(endOfWeek.Year(), endOfWeek.Month(), endOfWeek.Day(),
            23, 59, 59, 999999999, endOfWeek.Location())
    case "startofmonth":
        now := time.Now()
        return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
    case "endofmonth":
        now := time.Now()
        return time.Date(now.Year(), now.Month()+1, 0,
            23, 59, 59, 999999999, now.Location())
    }
    return nil
}
```

---

## 8. prefixCustomFields 集成

```go
// internal/api/handlers.go:1775-1833
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
            }
        case "$not":
            // 递归处理 NOT
            if vm, ok := v.(bson.M); ok {
                result[k] = prefixCustomFields(vm)
            }
        default:
            // 非系统字段、非 MongoDB 操作符 → 添加前缀
            if !systemFieldNames[k] && len(k) > 0 && k[0] != '$' {
                result["custom_fields."+k] = v
            } else {
                result[k] = v
            }
        }
    }
    return result
}
```

---

## 9. 单元测试示例

```go
// pkg/jql/parser_test.go
func TestParseQuery_BasicEquality(t *testing.T) {
    result, err := ParseQuery(`status = "active"`)
    assert.NoError(t, err)
    assert.Equal(t, bson.M{"status": "active"}, result)
}

func TestParseQuery_AndCondition(t *testing.T) {
    result, err := ParseQuery(`status = "active" AND price > 100`)
    assert.NoError(t, err)
    assert.Equal(t, bson.M{
        "$and": []bson.M{
            {"status": "active"},
            {"price": bson.M{"$gt": 100}},
        },
    }, result)
}
```

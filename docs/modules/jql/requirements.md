# JQL 查询模块 - 需求设计文档

## 1. 需求背景

数据中心系统的业务数据存储在 MongoDB 中，用户需要对数据进行灵活的查询过滤。MongoDB 原生的查询语法（bson.M）学习曲线较高，不适合非技术用户使用。因此设计 JQL (JSON Query Language) —— 一套类 SQL 的简洁查询语法，降低查询门槛。

## 2. 功能需求

### FR-JQL-01: 基本比较查询

支持标准比较运算符。

| 运算符 | 含义 | 示例 |
|--------|------|------|
| `=` | 等于 | `status = "active"` |
| `!=` | 不等于 | `status != "deleted"` |
| `>` | 大于 | `price > 100` |
| `<` | 小于 | `price < 50` |
| `>=` | 大于等于 | `age >= 18` |
| `<=` | 小于等于 | `age <= 65` |
| `~` | 模糊匹配（正则） | `name ~ "产品"` |

### FR-JQL-02: 列表匹配

| 语法 | 含义 | 示例 |
|------|------|------|
| `IN (...)` | 值在列表中 | `status IN ("active", "pending")` |
| `NOT IN (...)` | 值不在列表中 | `category NOT IN ("deleted", "archived")` |

### FR-JQL-03: 空值检查

| 语法 | 含义 | 示例 |
|------|------|------|
| `IS NULL` | 字段不存在或为空 | `assignee IS NULL` |
| `IS NOT NULL` | 字段存在且不为空 | `email IS NOT NULL` |

### FR-JQL-04: 逻辑运算

| 运算符 | 含义 | 优先级 |
|--------|------|--------|
| `NOT` | 逻辑非 | 最高 |
| `AND` | 逻辑与 | 中 |
| `OR` | 逻辑或 | 最低 |

### FR-JQL-05: 括号分组

支持使用括号改变运算优先级。

```
(status = "active") AND (price > 100 OR price < 50)
```

### FR-JQL-06: 内置函数

提供常用时间函数，简化时间范围查询。

| 函数 | 说明 |
|------|------|
| `Now()` | 当前时间 |
| `StartOfDay()` | 当天零点 |
| `EndOfDay()` | 当天 23:59:59 |
| `StartOfWeek()` | 本周一零点 |
| `EndOfWeek()` | 本周日 23:59:59 |
| `StartOfMonth()` | 本月 1 日零点 |
| `EndOfMonth()` | 本月最后一天 23:59:59 |

### FR-JQL-07: 值类型

自动识别和转换值类型：

| 值模式 | 类型 |
|--------|------|
| `"abc"` 或 `'abc'` | string |
| `123` | int |
| `3.14` | float64 |
| `true` / `false` | bool |

---

## 3. 语法规范

### 3.1 关键字（大小写不敏感）

```
AND, OR, NOT, IN, NOT IN, IS NULL, IS NOT NULL
```

### 3.2 运算符

```
=, !=, >, <, >=, <=, ~
```

### 3.3 分隔符

```
(, ), ,
```

### 3.4 字段名

```
字母/数字/下划线/点号
```

### 3.5 字符串值

```
单引号或双引号包裹
```

### 3.6 BNF 语法

```
expr       := or_expr
or_expr    := and_expr ("OR" and_expr)*
and_expr   := not_expr ("AND" not_expr)*
not_expr   := "NOT"? primary
primary    := "(" expr ")" | condition
condition  := field op value
            | field "IN" "(" values ")"
            | field "NOT IN" "(" values ")"
            | field "IS NULL"
            | field "IS NOT NULL"

op         := "=" | "!=" | ">" | "<" | ">=" | "<=" | "~"
values     := value ("," value)*
value      := STRING | NUMBER | BOOLEAN | FUNCTION
field      := IDENTIFIER
```

---

## 4. 查询示例

### 简单查询

```jql
status = "active"
price > 100
name ~ "产品"
```

### 条件组合

```jql
status = "active" AND price > 100
name = "A" OR name = "B"
NOT status = "deleted"
```

### 括号分组

```jql
(status = "active" OR status = "pending") AND price > 100
```

### 列表与空值

```jql
status IN ("active", "pending", "review")
category NOT IN ("deleted", "archived")
assignee IS NULL
email IS NOT NULL
```

### 时间函数

```jql
created > StartOfWeek() AND module = "movie"
updated < EndOfMonth() AND status NOT IN ("deleted", "archived")
```

---

## 5. 非功能需求

| 编号 | 需求 | 说明 |
|------|------|------|
| NFR-JQL-01 | 解析性能 | 单次解析在毫秒级完成 |
| NFR-JQL-02 | 语法容错 | 解析失败返回明确错误信息 |
| NFR-JQL-03 | 大小写不敏感 | 关键字大小写均可 |
| NFR-JQL-04 | MongoDB 兼容 | 输出标准 bson.M，直接用于查询 |

---

## 6. 错误处理

| 场景 | 错误消息 |
|------|----------|
| 未闭合的引号 | `unclosed string` |
| 未闭合的括号 | `expected closing parenthesis` |
| 字段后缺少操作符 | `expected operator after field` |
| 操作符后缺少值 | `expected value after operator` |
| 意外结束 | `unexpected end of input` |
| 无法识别的字符 | `unexpected character: X` |
| IN 后缺少括号 | `expected ( after IN/NOT IN` |

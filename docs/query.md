# 查询功能

## 1. JQL查询

### 1.1 基本概念

实现类JIRA JQL查询语句解析器，支持"字段 运算符 值/函数"格式

### 1.2 语法格式

```
field operator value
field operator function()
```

### 1.3 支持的运算符

| 运算符 | 描述 | 示例 |
|--------|------|------|
| = | 等于 | status = "open" |
| != | 不等于 | status != "closed" |
| > | 大于 | created > "2024-01-01" |
| < | 小于 | created < "2024-12-31" |
| >= | 大于等于 | priority >= 3 |
| <= | 小于等于 | priority <= 5 |
| IN | 在列表中 | status IN ("open", "pending") |
| NOT IN | 不在列表中 | status NOT IN ("deleted") |
| LIKE | 模糊匹配 | title LIKE "%keyword%" |
| IS NULL | 为空 | assignee IS NULL |
| IS NOT NULL | 不为空 | assignee IS NOT NULL |
| AND | 逻辑与 | status = "open" AND priority > 3 |
| OR | 逻辑或 | status = "open" OR status = "pending" |

### 1.4 内置函数

#### 用户相关函数

- **CurrentUser()**：获取当前用户
  - 示例：`assignee = CurrentUser()`

#### 时间相关函数

- **Now()**：获取当前时间
  - 示例：`created > Now()`

- **StartOfDay()**：获取当天开始时间
  - 示例：`created > StartOfDay()`

- **EndOfDay()**：获取当天结束时间
  - 示例：`created < EndOfDay()`

- **StartOfWeek()**：获取本周开始时间
  - 示例：`created > StartOfWeek()`

- **EndOfWeek()**：获取本周结束时间
  - 示例：`created < EndOfWeek()`

- **StartOfMonth()**：获取本月开始时间
  - 示例：`created > StartOfMonth()`

- **EndOfMonth()**：获取本月结束时间
  - 示例：`created < EndOfMonth()`

### 1.5 JQL示例

```
# 查询指定用户的待处理任务
status = "pending" AND assignee = CurrentUser()

# 查询最近一周创建的数据
created > StartOfWeek() AND module = "task"

# 查询高优先级且未完成的数据
priority >= 4 AND status NOT IN ("closed", "cancelled")

# 查询包含关键字的标题
title LIKE "%重要%" AND status != "deleted"
```

## 2. 转换逻辑

### 2.1 JQL到MongoDB转换

JQL解析器将JQL语句转换为MongoDB查询语句：

| JQL运算符 | MongoDB运算符 |
|-----------|--------------|
| = | $eq |
| != | $ne |
| > | $gt |
| < | $lt |
| >= | $gte |
| <= | $lte |
| IN | $in |
| NOT IN | $nin |
| LIKE | $regex |
| IS NULL | $exists: false |
| IS NOT NULL | $exists: true |
| AND | $and |
| OR | $or |

### 2.2 转换示例

**JQL:**
```
status = "open" AND priority >= 3
```

**MongoDB:**
```json
{
  "$and": [
    {"status": {"$eq": "open"}},
    {"priority": {"$gte": 3}}
  ]
}
```

## 3. 性能优化

### 3.1 索引策略

- 为常用查询字段创建索引
- 复合索引用于多条件查询
- 避免在索引字段上使用函数

### 3.2 查询优化

- 限制返回字段
- 使用分页避免大结果集
- 避免全表扫描

### 3.3 缓存策略

- 热门查询结果缓存
- 缓存失效策略

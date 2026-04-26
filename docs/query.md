# 查询功能

## 1. JQL查询

### 1.1 基本概念

实现类JIRA JQL查询语句解析器，支持"字段 运算符 值/函数"格式，提供安全的查询能力，防止SQL注入等攻击。

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

## 3. 安全考虑

### 3.1 防止注入攻击

- **参数化查询**：所有用户输入都通过参数化处理，防止注入攻击
- **字段白名单**：只允许查询预定义的字段，防止查询敏感字段
- **值类型验证**：验证所有输入值的类型，确保符合预期
- **运算符限制**：只允许使用预定义的运算符，防止恶意操作

### 3.2 防止DoS攻击

- **查询复杂度限制**：限制查询的嵌套层级和条件数量
- **结果集大小限制**：强制使用分页，防止返回过大的结果集
- **超时机制**：设置查询超时，防止长时间运行的查询

### 3.3 防止权限绕过

- **字段权限检查**：确保用户只能查询有权限的字段
- **数据权限检查**：确保用户只能查询有权限的数据
- **查询审计**：记录所有查询操作，便于安全审计

## 4. 性能优化

### 4.1 索引策略

- 为常用查询字段创建索引
- 复合索引用于多条件查询
- 避免在索引字段上使用函数

### 4.2 查询优化

- 限制返回字段
- 使用分页避免大结果集
- 避免全表扫描

### 4.3 缓存策略

- 热门查询结果缓存
- 缓存失效策略

## 5. 实现细节

### 5.1 解析器实现

- **词法分析**：将JQL语句分解为词法单元
- **语法分析**：构建抽象语法树
- **语义分析**：验证查询语义
- **查询转换**：转换为MongoDB查询

### 5.2 安全验证

- **字段验证**：验证字段是否存在且用户有权限
- **值验证**：验证值的类型和范围
- **查询验证**：验证查询复杂度和安全性

### 5.3 错误处理

- **语法错误**：返回详细的语法错误信息
- **语义错误**：返回字段或值的验证错误
- **安全错误**：返回权限或安全相关错误

## 6. API接口

### 6.1 查询接口

**GET /api/business/module/:module?jql=查询语句&page=1&pageSize=10**

**参数：**
- `jql`：JQL查询语句
- `page`：页码
- `pageSize`：每页数量

**响应：**
```json
{
  "data": [
    {
      "_id": "id",
      "module": "module",
      "description": "描述",
      "custom_fields": {...}
    }
  ],
  "total": 100,
  "page": 1,
  "pageSize": 10
}
```

### 6.2 语法验证接口

**POST /api/query/validate**

**请求体：**
```json
{
  "jql": "status = \"open\" AND priority >= 3"
}
```

**响应：**
```json
{
  "valid": true,
  "error": null
}
```

## 7. 使用示例

### 7.1 基本查询

```javascript
// 前端查询示例
const response = await fetch('/api/business/module/movie?jql=title LIKE "%harry%" AND year > 2000&page=1&pageSize=10', {
  headers: {
    'Authorization': 'Bearer ' + token
  }
});

const data = await response.json();
```

### 7.2 复杂查询

```javascript
// 复杂查询示例
const jql = 'status IN ("success", "failed") AND created > StartOfWeek() AND module = "movie"';
const response = await fetch(`/api/business/module/movie?jql=${encodeURIComponent(jql)}&page=1&pageSize=10`, {
  headers: {
    'Authorization': 'Bearer ' + token
  }
});
```

### 7.3 语法验证

```javascript
// 语法验证示例
const response = await fetch('/api/query/validate', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer ' + token
  },
  body: JSON.stringify({
    jql: 'status = "open" AND priority >= 3'
  })
});

const result = await response.json();
if (result.valid) {
  // 执行查询
} else {
  // 显示错误
  console.error(result.error);
}
```

## 8. 最佳实践

### 8.1 查询优化

- **使用索引字段**：优先使用有索引的字段进行查询
- **限制结果集**：始终使用分页，避免返回过多数据
- **简化查询**：分解复杂查询为多个简单查询
- **避免全表扫描**：使用索引字段进行过滤

### 8.2 安全最佳实践

- **验证输入**：始终验证用户输入的JQL语句
- **限制查询复杂度**：避免过于复杂的查询
- **监控查询**：监控慢查询和异常查询
- **权限检查**：确保用户只能查询有权限的数据

### 8.3 性能最佳实践

- **缓存热门查询**：缓存频繁执行的查询结果
- **批量查询**：合并多个小查询为一个大查询
- **异步查询**：对于复杂查询使用异步处理
- **预编译查询**：预编译常用查询模板

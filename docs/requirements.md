# 数据中心系统需求设计文档

## 1. 文档结构

本文档是数据中心系统的需求总览文档，详细规格被拆分为以下模块文档：

| 模块文档 | 描述 |
|----------|------|
| [rbac.md](rbac.md) | 用户权限系统 - 角色、权限、用户认证和授权 |
| [logging.md](logging.md) | 日志系统 - HTTP日志、应用日志、审计日志 |
| [business-data.md](business-data.md) | 业务数据管理 - 字段定义、业务数据、软删除 |
| [authentication.md](authentication.md) | 认证与授权 - JWT Token、登录、权限验证 |
| [query.md](query.md) | 查询功能 - JQL查询语法、转换逻辑、性能优化 |

## 2. 系统架构概览

### 2.1 技术栈

- **后端框架**：Gin (Go)
- **数据库**：MongoDB
- **日志**：Zerolog + Lumberjack
- **认证**：JWT Token

### 2.2 核心模块

```
datacenter/
├── cmd/server/          # 应用入口
├── internal/
│   ├── api/             # HTTP处理器
│   ├── middleware/      # 中间件（认证、日志）
│   ├── models/         # 数据模型
│   ├── storage/        # 数据存储层
│   └── rbac/           # 权限控制核心
├── pkg/
│   ├── auth/           # 认证相关
│   ├── rbac/           # RBAC服务
│   └── query/          # JQL查询解析
└── docs/               # 文档
```

## 3. 数据库设计 - 嵌入方案B

### 3.1 设计特点

RBAC系统采用嵌入方案B进行多对多关系设计：
- 用户文档中直接存储角色ID数组 (`role_ids`)
- 角色文档中直接存储权限ID数组 (`permission_ids`)
- 使用MongoDB的`$addToSet`和`$pull`操作进行数组元素的增删

### 3.2 实体关系图

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   User      │       │    Role     │       │ Permission  │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ _id         │       │ _id         │       │ _id         │
│ username    │       │ name        │       │ name        │
│ password    │       │ code        │       │ code        │
│ email       │       │ description │       │ description │
│ role_ids[]  │──────▶│ permission_ │──────▶│ created_by  │
│ (嵌入角色ID) │       │   ids[]     │       │ created_at  │
│ created_by  │       │ (嵌入权限ID) │       │ updated_by  │
│ created_at  │       │ created_by  │       │ updated_at  │
│ updated_by  │       │ created_at  │       └─────────────┘
│ updated_at  │       │ updated_by  │
└─────────────┘       │ updated_at  │
                      └─────────────┘

关系说明：
- User : Role = N : M （通过User.role_ids数组实现）
- Role : Permission = N : M （通过Role.permission_ids数组实现）
```

## 4. API 规范

### 4.1 通用响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

### 4.2 错误响应格式

```json
{
  "code": 400,
  "message": "error message"
}
```

## 5. 错误处理策略

### 5.1 错误分类

| 错误类型 | 状态码 | 描述 |
|----------|--------|------|
| 认证错误 | 401 | 未认证或Token无效 |
| 授权错误 | 403 | 无权限访问 |
| 参数错误 | 400 | 请求参数无效 |
| 资源不存在 | 404 | 请求资源不存在 |
| 服务器错误 | 500 | 服务器内部错误 |

### 5.2 错误处理流程

1. 捕获错误：在各个层级捕获可能的错误
2. 记录错误：使用日志系统记录错误详情
3. 转换错误：将内部错误转换为用户友好的错误信息
4. 返回错误：根据错误类型返回相应的HTTP状态码和错误信息

## 6. 验证规则

### 6.1 用户验证规则

- 用户名唯一，不能重复
- 邮箱格式正确，唯一
- 密码至少8位，使用bcrypt加密
- 用户可以同时拥有多个角色

### 6.2 角色验证规则

- 角色代码唯一，不能重复
- 角色名称必填

### 6.3 权限验证规则

- 权限代码唯一，不能重复
- 权限名称必填

### 6.4 关联验证规则 - 嵌入方案B

- 同一用户-角色组合不能重复分配（通过$addToSet保证）
- 同一角色-权限组合不能重复分配（通过$addToSet保证）
- 删除角色时，需要从所有用户的role_ids数组中移除该角色ID
- 删除权限时，需要从所有角色的permission_ids数组中移除该权限ID

## 7. 异常日志

- 记录异常发生的时间、位置、原因和上下文信息
- 对于关键操作的异常，发送告警通知
- 定期分析异常日志，优化系统稳定性

# 认证与授权

## 1. JWT Token

### 1.1 基本要求

- 实现基于JWT Token的前后端长连接认证机制
- Token需包含以下信息：
  - 用户身份标识
  - 用户角色信息
  - 权限范围

### 1.2 Token 结构

```json
{
  "user_id": "xxx",
  "username": "xxx",
  "roles": ["admin", "user"],
  "permissions": ["read", "write"],
  "exp": 1234567890,
  "iat": 1234567890
}
```

### 1.3 Token 刷新机制

- Access Token 有效期：15分钟
- Refresh Token 有效期：7天
- 当 Access Token 过期时，使用 Refresh Token 获取新的 Access Token

## 2. 认证接口

### 2.1 登录接口

- **POST /api/auth/login**：用户登录
  - 请求体：`{"username": "...", "password": "..."}`
  - 响应：Token和用户信息

### 2.2 刷新Token接口

- **POST /api/auth/refresh**：刷新Token
  - 请求头：`Authorization: Bearer <refresh_token>`
  - 响应：新的Access Token

### 2.3 登出接口

- **POST /api/auth/logout**：用户登出
  - 请求头：`Authorization: Bearer <token>`

## 3. 权限验证

### 3.1 中间件

实现权限验证中间件，保护API端点：
- 验证Token有效性
- 提取用户角色和权限
- 检查目标资源权限

### 3.2 验证流程

```
1. 检查请求头中的Token
2. 验证Token签名和有效期
3. 提取用户信息和权限
4. 检查目标端点所需权限
5. 允许或拒绝请求
```

## 4. 错误处理

### 4.1 认证错误

| 错误类型 | 状态码 | 描述 | 处理方式 |
|----------|--------|------|----------|
| 未认证 | 401 | 未提供Token或Token无效 | 返回认证错误信息，引导用户重新登录 |
| Token过期 | 401 | Token已过期 | 返回Token过期信息，引导用户刷新Token |
| 无权限 | 403 | 无权限访问 | 返回授权错误信息，提示用户联系管理员 |

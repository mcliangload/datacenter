# 认证模块 - 需求设计文档

## 1. 需求背景

数据中心系统需要一套安全的用户认证机制，确保只有经过身份验证的用户才能访问系统资源。考虑到系统面向运维与开发人员，认证方案需要兼顾安全性和易用性。

## 2. 功能需求

### FR-AUTH-01: 用户登录

**描述**: 用户通过用户名和密码登录系统，获取访问令牌。

**输入**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 明文密码 |

**输出**:
| 字段 | 类型 | 说明 |
|------|------|------|
| token | string | JWT 访问令牌 |
| user.id | string | 用户 ID |
| user.username | string | 用户名 |
| user.email | string | 邮箱 |
| user.roles | []string | 角色 ID 列表 |

**校验规则**:
- 用户名必须存在
- 密码必须匹配（bcrypt 比对）
- 用户名不存在或密码错误 → 返回 401 "Invalid credentials"

### FR-AUTH-02: 用户注册

**描述**: 新用户注册系统账户。

**输入**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 明文密码 |
| email | string | 是 | 邮箱 |

**规则**:
- 密码在存储前使用 bcrypt 哈希
- 注册时 role_ids 默认为空数组
- 返回 201 Created

### FR-AUTH-03: Token 验证

**描述**: 所有受保护 API 端点在处理请求前验证 JWT Token。

**规则**:
- Token 必须通过 `Authorization: Bearer <token>` 传递
- 验证签名、有效期
- 验证通过后将 user_id、roles、permissions 注入请求上下文
- 验证失败返回 401

### FR-AUTH-04: Token 刷新

**描述**: 允许在 Token 过期后的一段窗口时间内刷新 Token，无需重新登录。

**规则**:
- 过期 Token 可在刷新窗口内（默认 30 天）刷新
- 刷新后生成新 Token，保留原有用户信息和权限
- 超出刷新窗口需重新登录

## 3. 非功能需求

| 编号 | 需求 | 指标 |
|------|------|------|
| NFR-AUTH-01 | 密码安全 | bcrypt cost ≥ 10 |
| NFR-AUTH-02 | Token 签名安全 | HMAC-SHA256，密钥 ≥ 256 bit |
| NFR-AUTH-03 | Token 有效期 | 访问 Token ≤ 24h |
| NFR-AUTH-04 | 刷新窗口 | ≤ 30 天 |
| NFR-AUTH-05 | 密码最小长度 | ≥ 8 位 |
| NFR-AUTH-06 | 密码不可明文存储 | bcrypt 哈希 |
| NFR-AUTH-07 | 密码不可在响应中返回 | 所有 API 清除 password 字段 |

## 4. 业务流程

### 4.1 登录流程

```
用户输入用户名/密码
      │
      ▼
查询用户 (by username)
      │
      ├── 用户不存在 ──▶ 返回 401
      │
      ▼
bcrypt.CompareHashAndPassword
      │
      ├── 密码不匹配 ──▶ 返回 401
      │
      ▼
查询用户权限 (RBAC服务)
      │
      ▼
生成 JWT Token (HS256)
  ┌─ userID
  ├─ roles
  ├─ permissions
  ├─ exp = now + 24h
  └─ jti = UUID
      │
      ▼
返回 { token, user }
```

### 4.2 Token 刷新流程

```
请求携带过期 Token
      │
      ▼
尝试正常验证
      │
      ├── 未过期 ──▶ 生成新 Token
      │
      ▼
Token 已过期
      │
      ▼
手动解析 Claims（跳过过期检查）
      │
      ▼
检查: now < exp + refreshExpiration(720h) ?
      │
      ├── 是 ──▶ 生成新 Token（保留原有信息）
      │
      └── 否 ──▶ 返回 "refresh token expired"
```

## 5. 安全设计

### 5.1 认证方式选择

| 方案 | 优势 | 劣势 | 选择 |
|------|------|------|------|
| Session + Cookie | 服务端可控 | 有状态、扩展性差 | 否 |
| JWT | 无状态、扩展性好 | Token 无法主动失效 | **是** |
| OAuth 2.0 | 第三方集成 | 复杂、过度设计 | 否 |

### 5.2 密码策略

- 密码在服务端使用 bcrypt 单向哈希，不可逆
- 传输层依赖 HTTPS 加密
- 密码强度检查在后续迭代中添加

### 5.3 Token 安全

- 签名算法: HMAC-SHA256
- 密钥通过环境变量注入，不写入代码或配置文件
- JTI (JWT ID) 每次签发唯一，可用于未来的 Token 黑名单机制
- Token 不设置 `aud` (audience) 声明，简化单服务部署场景

## 6. 接口定义

### POST /api/auth/login

```
Request:
{
  "username": "admin",
  "password": "admin123"
}

Response 200:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "username": "admin",
    "email": "admin@example.com",
    "roles": ["role_id_1"]
  }
}

Response 401:
{
  "error": "Invalid credentials"
}
```

### POST /api/auth/register

```
Request:
{
  "username": "newuser",
  "password": "password123",
  "email": "newuser@example.com"
}

Response 201:
{
  "id": "507f1f77bcf86cd799439012",
  "username": "newuser",
  "email": "newuser@example.com"
}
```

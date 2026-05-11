# 认证模块 - 技术文档

## 1. 模块概述

认证模块负责整个系统的用户身份认证与令牌管理，基于 JWT (JSON Web Token) 实现无状态认证，使用 bcrypt 进行密码安全存储。

### 模块位置

```
internal/auth/
├── jwt.go            # JWT 服务（生成/验证/刷新）+ 密码工具
├── middleware.go      # Gin 认证中间件
└── test/
    └── jwt_test.go    # 单元测试
```

### 核心依赖

| 依赖 | 用途 |
|------|------|
| `github.com/golang-jwt/jwt/v5` | JWT 令牌生成与验证 |
| `github.com/google/uuid` | Token JTI (JWT ID) 唯一标识 |
| `golang.org/x/crypto/bcrypt` | 密码哈希与比对 |

---

## 2. 接口设计

### 2.1 JWTService 接口

```go
type JWTService interface {
    GenerateToken(userID string, roles, permissions []string) (string, error)
    ValidateToken(tokenString string) (*Claims, error)
    RefreshToken(tokenString string) (string, error)
}
```

### 2.2 Claims 结构

```go
type Claims struct {
    UserID      string   `json:"user_id"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
    jwt.RegisteredClaims
}
```

**RegisteredClaims 字段说明**:

| 字段 | 含义 | 取值 |
|------|------|------|
| `Issuer` | 签发方 | `"datacenter"` |
| `Subject` | 主体标识 | `userID` |
| `ID` | JTI 唯一标识 | `uuid.New().String()` |
| `IssuedAt` | 签发时间 | `time.Now()` |
| `NotBefore` | 生效时间 | `time.Now()` |
| `ExpiresAt` | 过期时间 | `time.Now() + tokenExpiration` |

---

## 3. 核心实现

### 3.1 构造函数

```go
func NewJWTService(secretKey string, tokenExpiration, refreshExpiration time.Duration) JWTService
```

参数从环境变量注入：
- `JWT_SECRET` → `secretKey`
- `JWT_EXPIRATION` (小时) → `tokenExpiration`（默认 24h）
- `JWT_REFRESH_EXPIRATION` (小时) → `refreshExpiration`（默认 720h）

### 3.2 GenerateToken

```go
func (s *jwtService) GenerateToken(userID string, roles, permissions []string) (string, error)
```

**流程**:
1. 构造 `Claims`，填充 userID/roles/permissions
2. 设置 RegisteredClaims (issuer=datacenter, subject=userID, jti=UUID)
3. 计算过期时间 = `time.Now() + tokenExpiration`
4. 使用 HMAC-SHA256 签名
5. 返回签名字符串

### 3.3 ValidateToken

```go
func (s *jwtService) ValidateToken(tokenString string) (*Claims, error)
```

**流程**:
1. `jwt.ParseWithClaims(tokenString, &Claims{}, keyFunc)`
2. keyFunc 返回 `[]byte(secretKey)` 用于签名验证
3. 检查 `token.Valid` 和类型断言
4. 验证通过返回 `*Claims`，失败返回 error

**可能的错误**:
- `jwt.ErrTokenExpired` — Token 已过期
- `jwt.ErrSignatureInvalid` — 签名不匹配
- `jwt.ErrTokenMalformed` — Token 格式错误

### 3.4 RefreshToken

```go
func (s *jwtService) RefreshToken(tokenString string) (string, error)
```

**流程**:
1. 先尝试正常验证（`ValidateToken`）
2. 如果 Token 未过期 → 直接生成新 Token
3. 如果 Token 已过期 (`jwt.ErrTokenExpired`)：
   - 手动解析 Claims（跳过过期检查）
   - 检查是否在刷新窗口内 (`ExpiresAt + refreshExpiration`)
   - 在窗口内 → 生成新 Token
   - 超出窗口 → 返回 `"refresh token expired"`
4. 新 Token 保留原 userID/roles/permissions

---

## 4. 密码工具

### 4.1 HashPassword

```go
func HashPassword(password string) (string, error)
```

使用 bcrypt 的 `GenerateFromPassword`，cost 为 `bcrypt.DefaultCost` (10)。

### 4.2 CheckPassword

```go
func CheckPassword(password, hash string) (string, error)
```

使用 bcrypt 的 `CompareHashAndPassword` 进行常量时间比较。

---

## 5. 认证中间件

### 5.1 AuthMiddleware

位于 `internal/api/handlers.go`，作为 `Handler` 的方法实现：

```go
func (h *Handler) AuthMiddleware() gin.HandlerFunc
```

**流程**:
1. 从 `Authorization` header 提取 `Bearer <token>`
2. 调用 `jwtService.ValidateToken(tokenString)`
3. 验证通过 → `c.Set("user_id", claims.UserID)`、`c.Set("roles", claims.Roles)`
4. 查询用户权限 → `c.Set("permissions", perms)`
5. 验证失败 → 返回 401

**Gin Context 注入的键**:
| Key | 类型 | 来源 |
|-----|------|------|
| `user_id` | string | Claims.UserID |
| `roles` | []string | Claims.Roles |
| `permissions` | []string | RBAC 服务查询 |

---

## 6. 配置参数

| 参数 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| 签名密钥 | `JWT_SECRET` | `your-secret-key` | HMAC-SHA256 密钥 |
| Token 有效期 | `JWT_EXPIRATION` | 24 (小时) | 访问 Token 有效期 |
| 刷新窗口 | `JWT_REFRESH_EXPIRATION` | 720 (小时) | Token 过期后可刷新的时间窗口 |

---

## 7. 安全考量

| 方面 | 措施 |
|------|------|
| 签名算法 | HMAC-SHA256 (HS256) |
| 密钥管理 | 通过环境变量注入，不入库 |
| 密码存储 | bcrypt 单向哈希（cost=10） |
| 密码传输 | 通过 HTTPS + JSON body |
| 密码响应 | 所有 API 响应中清除 `password` 字段 |
| Token 唯一性 | 每次签发使用 UUID 作为 JTI |
| 过期策略 | 短期 Token (24h) + 长期刷新窗口 (30d) |

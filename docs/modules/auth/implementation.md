# 认证模块 - 需求实现文档

## 1. 实现概述

认证模块的核心实现在 `internal/auth/jwt.go`，通过 `JWTService` 接口暴露令牌管理能力。中间件实现在 `internal/api/handlers.go` 的 `AuthMiddleware()` 方法中。

---

## 2. 文件清单

| 文件 | 说明 |
|------|------|
| `internal/auth/jwt.go` | JWT Token 生成/验证/刷新，密码哈希/校验 |
| `internal/auth/middleware.go` | (辅助) 认证中间件定义 |
| `internal/api/handlers.go` | `Login()`、`Register()`、`AuthMiddleware()` |
| `cmd/server/main.go` | JWT 服务初始化与依赖注入 |

---

## 3. 初始化代码

### 3.1 main.go 中的初始化

```go
// cmd/server/main.go:72-76
jwtService := auth.NewJWTService(
    getEnv("JWT_SECRET", "your-secret-key"),
    time.Duration(getEnvAsInt("JWT_EXPIRATION", 24))*time.Hour,
    time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRATION", 720))*time.Hour,
)
```

### 3.2 Handler 中的依赖注入

```go
// internal/api/handlers.go:33-43
type Handler struct {
    storage               storage.Storage
    rbacStorage           storage.RBACStorage
    scraper               scraper.Scraper
    jwtService            auth.JWTService      // JWT 服务接口
    rbacService           *rbac.Service
    collectionRBACStorage storage.CollectionRBACStorage
    collectionRBACService *rbac.CollectionRBACService
}
```

---

## 4. 登录实现

```go
// internal/api/handlers.go:183-221
func (h *Handler) Login(c *gin.Context) {
    // 1. 解析请求体
    var req struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 2. 查询用户
    user, err := h.rbacStorage.GetUserByUsername(req.Username)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }

    // 3. 验证密码 (bcrypt)
    if err := auth.CheckPassword(req.Password, user.Password); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }

    // 4. 获取用户权限
    perms, _ := h.rbacService.GetUserPermissions(context.Background(), user.ID.Hex())

    // 5. 生成 JWT Token
    token, err := h.jwtService.GenerateToken(user.ID.Hex(), user.RoleIDs, perms)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    // 6. 返回 Token + 用户信息
    c.JSON(http.StatusOK, gin.H{
        "token": token,
        "user": gin.H{
            "id": user.ID, "username": user.Username,
            "email": user.Email, "roles": user.RoleIDs,
        },
    })
}
```

**关键点**: 登录后将用户权限编码进 Token，减少后续请求的权限查询开销。但中间件仍会重新查询权限以确保实时性。

---

## 5. Token 生成实现

```go
// internal/auth/jwt.go:44-66
func (s *jwtService) GenerateToken(userID string, roles, permissions []string) (string, error) {
    claims := &Claims{
        UserID:      userID,
        Roles:       roles,
        Permissions: permissions,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "datacenter",
            Subject:   userID,
            ID:        uuid.New().String(),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.secretKey))
}
```

---

## 6. Token 验证与刷新实现

### 6.1 验证

```go
// internal/auth/jwt.go:69-83
func (s *jwtService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(s.secretKey), nil
    })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    return nil, errors.New("invalid token")
}
```

### 6.2 刷新

```go
// internal/auth/jwt.go:86-111
func (s *jwtService) RefreshToken(tokenString string) (string, error) {
    claims, err := s.ValidateToken(tokenString)
    if err != nil {
        // 允许过期 Token 刷新
        if errors.Is(err, jwt.ErrTokenExpired) {
            token, parseErr := jwt.ParseWithClaims(tokenString, &Claims{}, ...)
            if claims, ok := token.Claims.(*Claims); ok {
                // 检查刷新窗口
                if time.Now().After(claims.ExpiresAt.Time.Add(s.refreshExpiration)) {
                    return "", errors.New("refresh token expired")
                }
                return s.GenerateToken(claims.UserID, claims.Roles, claims.Permissions)
            }
        }
        return "", err
    }
    return s.GenerateToken(claims.UserID, claims.Roles, claims.Permissions)
}
```

**关键设计**: 过期 Token 的刷新使用 `jwt.ParseWithClaims` 二次解析（跳过 `Valid` 检查），通过 `ExpiresAt.Add(refreshExpiration)` 判断是否在刷新窗口内。

---

## 7. 认证中间件实现

```go
// internal/api/handlers.go:260-293
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 提取 Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        // 2. 移除 "Bearer " 前缀
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
            c.Abort()
            return
        }

        // 3. 验证 Token
        claims, err := h.jwtService.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // 4. 注入上下文
        c.Set("user_id", claims.UserID)
        c.Set("roles", claims.Roles)

        // 5. 重新查询权限（确保实时性）
        perms, err := h.rbacService.GetUserPermissions(context.Background(), claims.UserID)
        if err == nil {
            c.Set("permissions", perms)
        }

        c.Next()
    }
}
```

---

## 8. 密码工具实现

```go
// internal/auth/jwt.go:114-125
func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

func CheckPassword(password, hash string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
```

---

## 9. 路由注册

```go
// internal/api/handlers.go:46-47
// 公开路由（无需认证）
r.POST("/api/auth/login", h.Login)
r.POST("/api/auth/register", h.Register)

// 受保护路由组
protected := r.Group("/api")
protected.Use(h.AuthMiddleware())  // 所有 /api/* 路由需要 JWT 认证
```

---

## 10. 错误处理

| 场景 | HTTP 状态码 | 错误消息 |
|------|------------|----------|
| 请求体解析失败 | 400 | `err.Error()` |
| 用户名不存在 | 401 | `"Invalid credentials"` |
| 密码不匹配 | 401 | `"Invalid credentials"` |
| Token 生成失败 | 500 | `"Failed to generate token"` |
| 缺少 Authorization header | 401 | `"Authorization header required"` |
| 非 Bearer Token | 401 | `"Bearer token required"` |
| Token 无效/过期 | 401 | `"Invalid token"` |
| 刷新超出窗口 | 401 | `"refresh token expired"` |

package middleware

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"datacenter/internal/config"
	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/response"
	"datacenter/internal/store"
)

// gin.Context 中注入的用户信息 key
const (
	ctxKeyUserID   = "auth_user_id"
	ctxKeyUsername = "auth_username"
	ctxKeyRole     = "auth_role"
)

// Auth JWT 鉴权中间件：校验 Authorization: Bearer <token> 并注入用户信息。
// v0.15 加固：JWT 校验后**查库校验用户存活性**（存在且 active），并采用**实时角色**
// 覆盖 token 载荷——彻底杜绝"幽灵身份"：用户被删除/重建/禁用/降权后，旧 token
// 直接 401 失效，不会出现"界面显示 admin 但权限判定按普通用户"的权限断裂。
func Auth(jwtCfg config.JWTConfig, users *store.UserStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractBearerToken(c)
		if tokenStr == "" {
			response.AbortWithError(c, errno.ErrUnauthorized)
			return
		}

		claims, err := parseToken(tokenStr, jwtCfg)
		if err != nil {
			response.AbortWithError(c, errno.ErrUnauthorized.WithCause(err))
			return
		}

		// 用户存活性校验：不存在/被禁用 → 401（强制重新登录）
		uid, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			response.AbortWithError(c, errno.ErrUnauthorized.WithCause(err))
			return
		}
		u, err := users.FindByID(c.Request.Context(), uid)
		if err != nil {
			response.AbortWithError(c, errno.ErrInternal.WithCause(err))
			return
		}
		if u == nil || u.Status != model.UserStatusActive {
			response.AbortWithError(c, errno.ErrUnauthorized)
			return
		}
		// 安全增强 P1-7：改密后旧 token 立即失效（pwd_ver 不匹配）
		if claims.PwdVer != u.PasswordVersion {
			response.AbortWithError(c, errno.ErrUnauthorized)
			return
		}

		c.Set(ctxKeyUserID, u.ID.Hex())
		c.Set(ctxKeyUsername, u.Username)
		// 实时角色优先：角色变更即时生效，避免旧 token 携带过期角色
		c.Set(ctxKeyRole, u.Role)
		c.Next()
	}
}

// RequireRole 全局角色校验（用于公共权限，如 admin 专属接口）
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentRole(c) != role {
			response.AbortWithError(c, errno.ErrForbidden)
			return
		}
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}

func parseToken(tokenStr string, jwtCfg config.JWTConfig) (*model.Claims, error) {
	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(jwtCfg.Secret), nil
		},
		jwt.WithIssuer(jwtCfg.Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// CurrentUserID 从上下文取当前用户 ID
func CurrentUserID(c *gin.Context) string {
	return c.GetString(ctxKeyUserID)
}

// CurrentUsername 从上下文取当前用户名
func CurrentUsername(c *gin.Context) string {
	return c.GetString(ctxKeyUsername)
}

// CurrentRole 从上下文取当前用户全局角色
func CurrentRole(c *gin.Context) string {
	return c.GetString(ctxKeyRole)
}

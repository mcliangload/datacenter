package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"datacenter/internal/config"
	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// AuthService 认证服务
type AuthService struct {
	users  *store.UserStore
	audit  *store.AuditStore
	jwtCfg config.JWTConfig
	guard  *loginGuard // 安全增强 P0-1：登录防爆破
}

// NewAuthService 构造认证服务
func NewAuthService(users *store.UserStore, audit *store.AuditStore, jwtCfg config.JWTConfig) *AuthService {
	return &AuthService{users: users, audit: audit, jwtCfg: jwtCfg, guard: newLoginGuard()}
}

// Login 校验用户名密码并签发 JWT，成功返回 (token, user, nil)。
// clientIP 来自请求对端（用于防爆破计数；反向代理场景需配置 TrustedProxies）。
// 锁定状态与密码错误返回同一文案（防账号枚举，P0-1）。
func (s *AuthService) Login(ctx context.Context, clientIP, username, password string) (string, *model.User, *errno.Error) {
	key := clientIP + "|" + username
	if !s.guard.allow(key) {
		// 锁定窗口内：文案与密码错误一致，不泄露锁定状态
		return "", nil, errno.New(errno.ErrUnauthorized.Code, errno.ErrUnauthorized.HTTPStatus, "用户名或密码错误")
	}
	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return "", nil, errno.ErrInternal.WithCause(err)
	}
	if u == nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		// 不区分「用户不存在」与「密码错误」，避免账号枚举
		s.guard.fail(key)
		s.audit.Log(ctx, primitive.NilObjectID, "auth.login_fail",
			"登录失败: "+username+" (ip="+clientIP+")", nil, nil)
		return "", nil, errno.New(errno.ErrUnauthorized.Code, errno.ErrUnauthorized.HTTPStatus, "用户名或密码错误").
			WithCause(errors.New("invalid username or password"))
	}
	if u.Status != model.UserStatusActive {
		s.guard.fail(key)
		s.audit.Log(ctx, u.ID, "auth.login_fail",
			"登录失败(已禁用): "+username+" (ip="+clientIP+")", nil, nil)
		return "", nil, errno.ErrForbidden.WithCause(errors.New("user disabled: " + u.Username))
	}

	s.guard.success(key)
	s.audit.Log(ctx, u.ID, "auth.login_success",
		"登录成功: "+username+" (ip="+clientIP+")", nil, nil)

	token, err := s.sign(u)
	if err != nil {
		return "", nil, errno.ErrInternal.WithCause(err)
	}
	return token, u, nil
}

// sign 为用户签发 JWT
func (s *AuthService) sign(u *model.User) (string, error) {
	now := time.Now()
	claims := &model.Claims{
		UserID:   u.ID.Hex(),
		Username: u.Username,
		Role:     u.Role,
		PwdVer:   u.PasswordVersion, // 安全增强 P1-7：改密吊销旧 token
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.jwtCfg.Issuer,
			Subject:   u.ID.Hex(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.jwtCfg.ExpireHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.Secret))
}

// ChangePassword 修改当前用户密码（个人设置）：校验原密码后更新
func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPass, newPass string) *errno.Error {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errno.ErrParam.WithCause(err)
	}
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if u == nil {
		return errno.ErrUserNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPass)) != nil {
		// 400 而非 401：用户已持有有效 token，避免前端将密码错误当作未认证强制登出
		return errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "原密码错误")
	}
	// 安全增强 P0-3：密码强度校验（≥8 位 + 字母 + 数字）
	if e := validatePasswordStrength(newPass); e != nil {
		return e
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	// 安全增强 P1-7：改密后吊销该用户已签发的全部旧 token
	if err := s.users.UpdateFields(ctx, uid, bson.M{
		"password_hash":    string(hash),
		"password_version": u.PasswordVersion + 1,
	}); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	return nil
}

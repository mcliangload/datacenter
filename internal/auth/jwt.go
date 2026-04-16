package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims JWT声明结构
type Claims struct {
	UserID      string   `json:"user_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// JWTService JWT服务接口
type JWTService interface {
	GenerateToken(userID string, roles, permissions []string) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(tokenString string) (string, error)
}

// jwtService JWT服务实现
type jwtService struct {
	secretKey         string
	tokenExpiration   time.Duration
	refreshExpiration time.Duration
}

// NewJWTService 创建JWT服务实例
func NewJWTService(secretKey string, tokenExpiration, refreshExpiration time.Duration) JWTService {
	return &jwtService{
		secretKey:         secretKey,
		tokenExpiration:   tokenExpiration,
		refreshExpiration: refreshExpiration,
	}
}

// GenerateToken 生成JWT Token
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
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken 验证JWT Token
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

// RefreshToken 刷新JWT Token
func (s *jwtService) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		// 允许过期的token进行刷新
		if errors.Is(err, jwt.ErrTokenExpired) {
			token, parseErr := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(s.secretKey), nil
			})
			if parseErr != nil {
				return "", parseErr
			}
			if claims, ok := token.Claims.(*Claims); ok {
				// 检查刷新过期时间
				if time.Now().After(claims.ExpiresAt.Add(s.refreshExpiration)) {
					return "", errors.New("refresh token expired")
				}
				// 生成新token
				return s.GenerateToken(claims.UserID, claims.Roles, claims.Permissions)
			}
		}
		return "", err
	}

	// 生成新token
	return s.GenerateToken(claims.UserID, claims.Roles, claims.Permissions)
}

// HashPassword 加密密码
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

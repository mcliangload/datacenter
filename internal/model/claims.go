package model

import "github.com/golang-jwt/jwt/v5"

// Claims 自定义 JWT 声明
type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	PwdVer   int    `json:"pwd_ver"` // 签发时的密码版本（安全增强 P1-7：改密后旧 token 立即失效；缺省为 0 兼容存量 token）
	jwt.RegisteredClaims
}

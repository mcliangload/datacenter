package handler

import (
	"github.com/gin-gonic/gin"

	"datacenter/internal/middleware"
	"datacenter/internal/response"
	"datacenter/internal/service"
)

// AuthHandler 认证相关接口
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler 构造认证处理器
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}

	token, user, err := h.auth.Login(c.Request.Context(), c.ClientIP(), req.Username, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"token": token, "user": user})
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	response.OK(c, gin.H{
		"id":       middleware.CurrentUserID(c),
		"username": middleware.CurrentUsername(c),
		"role":     middleware.CurrentRole(c),
	})
}

// Logout POST /api/v1/auth/logout
// JWT 为无状态方案，客户端丢弃 token 即可；如需黑名单/吊销在此扩展。
func (h *AuthHandler) Logout(c *gin.Context) {
	response.OK(c, nil)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword POST /api/v1/auth/password（个人设置：修改本人密码）
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.auth.ChangePassword(c.Request.Context(), middleware.CurrentUserID(c), req.OldPassword, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

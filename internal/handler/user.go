package handler

import (
	"github.com/gin-gonic/gin"

	"datacenter/internal/middleware"
	"datacenter/internal/response"
	"datacenter/internal/service"
)

// UserHandler 用户管理接口（全部 admin 专属）
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler 构造用户管理处理器
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Create POST /api/v1/users
func (h *UserHandler) Create(c *gin.Context) {
	var req service.CreateUserReq
	if !bindJSON(c, &req) {
		return
	}
	u, err := h.svc.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, u)
}

// List GET /api/v1/users
func (h *UserHandler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	users, total, err := h.svc.List(c.Request.Context(), page, pageSize, c.Query("keyword"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(users, total, page, pageSize))
}

// Update PATCH /api/v1/users/:id
func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req service.UpdateUserReq
	if !bindJSON(c, &req) {
		return
	}
	u, err := h.svc.Update(c.Request.Context(), middleware.CurrentUserID(c), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, u)
}

// Delete DELETE /api/v1/users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), middleware.CurrentUserID(c), id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

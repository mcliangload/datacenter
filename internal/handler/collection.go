package handler

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"datacenter/internal/errno"
	"datacenter/internal/middleware"
	"datacenter/internal/model"
	"datacenter/internal/response"
	"datacenter/internal/service"
)

// CollectionHandler 集合管理接口
type CollectionHandler struct {
	svc *service.CollectionService
}

// NewCollectionHandler 构造集合管理处理器
func NewCollectionHandler(svc *service.CollectionService) *CollectionHandler {
	return &CollectionHandler{svc: svc}
}

// Create POST /api/v1/collections（admin）
func (h *CollectionHandler) Create(c *gin.Context) {
	var req service.CreateCollectionReq
	if !bindJSON(c, &req) {
		return
	}
	actorID, _ := primitive.ObjectIDFromHex(middleware.CurrentUserID(c))
	col, err := h.svc.Create(c.Request.Context(), actorID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col)
}

// List GET /api/v1/collections
func (h *CollectionHandler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	isAdmin := middleware.CurrentRole(c) == model.RoleAdmin
	cols, total, err := h.svc.List(c.Request.Context(), middleware.CurrentUserID(c), isAdmin, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(cols, total, page, pageSize))
}

// Get GET /api/v1/collections/:id
func (h *CollectionHandler) Get(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	col, err := h.svc.Get(c.Request.Context(), middleware.CurrentUserID(c), id, middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col)
}

// UpdateMeta PATCH /api/v1/collections/:id（集合管理员）
func (h *CollectionHandler) UpdateMeta(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Description string `json:"description"`
	}
	if !bindJSON(c, &req) {
		return
	}
	col, err := h.svc.UpdateMeta(c.Request.Context(), middleware.CurrentUserID(c), id, req.Description)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col)
}

// Delete DELETE /api/v1/collections/:id（admin）
func (h *CollectionHandler) Delete(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// GetTags GET /api/v1/collections/:id/tags
func (h *CollectionHandler) GetTags(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	col, err := h.svc.Get(c.Request.Context(), middleware.CurrentUserID(c), id, middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col.TagSchema)
}

// PutTags PUT /api/v1/collections/:id/tags（集合管理员，全量替换）
func (h *CollectionHandler) PutTags(c *gin.Context) {
	h.updateTags(c, true)
}

// PatchTags PATCH /api/v1/collections/:id/tags（集合管理员，增量合并）
func (h *CollectionHandler) PatchTags(c *gin.Context) {
	h.updateTags(c, false)
}

func (h *CollectionHandler) updateTags(c *gin.Context, replace bool) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Tags []model.TagDefinition `json:"tags"`
	}
	if !bindJSON(c, &req) {
		return
	}
	col, err := h.svc.UpdateTagSchema(c.Request.Context(), middleware.CurrentUserID(c), id, req.Tags, replace)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col)
}

// PutScript PUT /api/v1/collections/:id/script（集合管理员）
func (h *CollectionHandler) PutScript(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if !bindJSON(c, &req) {
		return
	}
	col, err := h.svc.UpdateScrapeScript(c.Request.Context(), middleware.CurrentUserID(c), id, req.Path)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col)
}

// ListMembers GET /api/v1/collections/:id/members
func (h *CollectionHandler) ListMembers(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	col, err := h.svc.Get(c.Request.Context(), middleware.CurrentUserID(c), id, middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col.Members)
}

// GrantMember POST /api/v1/collections/:id/members（集合管理员授权操作工）
func (h *CollectionHandler) GrantMember(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if !bindJSON(c, &req) {
		return
	}
	targetID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		response.Error(c, errno.ErrParam.WithCause(err))
		return
	}
	col, e := h.svc.GrantMember(c.Request.Context(), middleware.CurrentUserID(c), id, targetID)
	if e != nil {
		response.Error(c, e)
		return
	}
	response.OK(c, col)
}

// RemoveMember DELETE /api/v1/collections/:id/members/:userId（集合管理员移除操作工）
func (h *CollectionHandler) RemoveMember(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	targetID, ok := parseObjectID(c, "userId")
	if !ok {
		return
	}
	col, err := h.svc.RemoveMember(c.Request.Context(), middleware.CurrentUserID(c), id, targetID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col)
}

// PutDeletePolicy PUT /api/v1/collections/:id/delete-policy（集合管理员配置删除策略）
func (h *CollectionHandler) PutDeletePolicy(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req model.DeletePolicy
	if !bindJSON(c, &req) {
		return
	}
	col, err := h.svc.UpdateDeletePolicy(c.Request.Context(), middleware.CurrentUserID(c), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, col)
}

// AssignAdmin PUT /api/v1/collections/:id/admin（admin 更换集合管理员）
func (h *CollectionHandler) AssignAdmin(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if !bindJSON(c, &req) {
		return
	}
	newAdminID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		response.Error(c, errno.ErrParam.WithCause(err))
		return
	}
	actorID, _ := primitive.ObjectIDFromHex(middleware.CurrentUserID(c))
	col, e := h.svc.AssignCollectionAdmin(c.Request.Context(), actorID, id, newAdminID)
	if e != nil {
		response.Error(c, e)
		return
	}
	response.OK(c, col)
}

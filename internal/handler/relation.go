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

// RelationHandler 数据项关联关系接口
type RelationHandler struct {
	svc *service.RelationService
}

// NewRelationHandler 构造关联关系处理器
func NewRelationHandler(svc *service.RelationService) *RelationHandler {
	return &RelationHandler{svc: svc}
}

// Create POST /api/v1/items/:itemId/relations
func (h *RelationHandler) Create(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	var req service.CreateRelationReq
	if !bindJSON(c, &req) {
		return
	}
	r, err := h.svc.Create(c.Request.Context(), middleware.CurrentUserID(c), itemID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}

// CreateBatch POST /api/v1/items/:itemId/relations/batch
func (h *RelationHandler) CreateBatch(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	var req struct {
		Relations []service.CreateRelationReq `json:"relations" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.svc.CreateBatch(c.Request.Context(), middleware.CurrentUserID(c), itemID, req.Relations)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, result)
}

// List GET /api/v1/items/:itemId/relations?direction=&type=&page=&page_size=
func (h *RelationHandler) List(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	page, pageSize := parsePagination(c)
	views, total, err := h.svc.List(c.Request.Context(), middleware.CurrentUserID(c),
		itemID, c.DefaultQuery("direction", "out"), c.Query("type"), page, pageSize,
		middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(views, total, page, pageSize))
}

// Tree GET /api/v1/items/:itemId/tree?direction=&depth=&type=
func (h *RelationHandler) Tree(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	depth := 3
	if v, err := parseIntQuery(c, "depth", 3); err == nil {
		depth = v
	}
	tree, err := h.svc.Tree(c.Request.Context(), middleware.CurrentUserID(c),
		itemID, c.DefaultQuery("direction", "desc"), depth, c.Query("type"),
		middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tree)
}

// Badges GET /api/v1/items/relation-badges?ids=a,b,c
func (h *RelationHandler) Badges(c *gin.Context) {
	raw := c.Query("ids")
	ids := make([]primitive.ObjectID, 0, 8)
	for _, part := range splitComma(raw) {
		id, err := primitive.ObjectIDFromHex(part)
		if err != nil {
			response.Error(c, errno.ErrParam.WithCause(err))
			return
		}
		ids = append(ids, id)
	}
	badges, err := h.svc.Badges(c.Request.Context(), middleware.CurrentUserID(c), ids)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, badges)
}

// UpdateMeta PATCH /api/v1/relations/:relationId
func (h *RelationHandler) UpdateMeta(c *gin.Context) {
	relationID, ok := parseObjectID(c, "relationId")
	if !ok {
		return
	}
	var req struct {
		Meta map[string]interface{} `json:"meta"`
	}
	if !bindJSON(c, &req) {
		return
	}
	r, err := h.svc.UpdateMeta(c.Request.Context(), middleware.CurrentUserID(c), relationID, req.Meta)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}

// Delete DELETE /api/v1/relations/:relationId
func (h *RelationHandler) Delete(c *gin.Context) {
	relationID, ok := parseObjectID(c, "relationId")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), middleware.CurrentUserID(c), relationID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// Impact GET /api/v1/items/:itemId/delete-impact（删除预检）
func (h *RelationHandler) Impact(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	impact, err := h.svc.Impact(c.Request.Context(), middleware.CurrentUserID(c), itemID,
		middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, impact)
}

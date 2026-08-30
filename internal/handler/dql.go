package handler

import (
	"github.com/gin-gonic/gin"

	"datacenter/internal/middleware"
	"datacenter/internal/model"
	"datacenter/internal/response"
	"datacenter/internal/service"
)

// DQLHandler 数据查询（DQL）接口
type DQLHandler struct {
	svc *service.DQLService
}

// NewDQLHandler 构造 DQL 处理器
func NewDQLHandler(svc *service.DQLService) *DQLHandler {
	return &DQLHandler{svc: svc}
}

type dqlQueryRequest struct {
	DQL      string `json:"dql" binding:"required"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// Query POST /api/v1/dql/query
func (h *DQLHandler) Query(c *gin.Context) {
	var req dqlQueryRequest
	if !bindJSON(c, &req) {
		return
	}
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	items, total, err := h.svc.Query(c.Request.Context(),
		middleware.CurrentUserID(c),
		middleware.CurrentRole(c) == model.RoleAdmin,
		req.DQL, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(items, total, page, pageSize))
}

// Aggregate POST /api/v1/dql/aggregate（系统优化 1.2：分组统计）
func (h *DQLHandler) Aggregate(c *gin.Context) {
	var req service.AggregateReq
	if !bindJSON(c, &req) {
		return
	}
	results, err := h.svc.Aggregate(c.Request.Context(),
		middleware.CurrentUserID(c),
		middleware.CurrentRole(c) == model.RoleAdmin,
		req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, results)
}

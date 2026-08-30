package handler

import (
	"github.com/gin-gonic/gin"

	"datacenter/internal/middleware"
	"datacenter/internal/model"
	"datacenter/internal/response"
	"datacenter/internal/service"
)

// StatsHandler 仪表盘统计接口
type StatsHandler struct {
	svc *service.StatsService
}

// NewStatsHandler 构造统计处理器
func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// Overview GET /api/v1/stats/overview
func (h *StatsHandler) Overview(c *gin.Context) {
	ov, err := h.svc.Overview(c.Request.Context(),
		middleware.CurrentUserID(c),
		middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ov)
}

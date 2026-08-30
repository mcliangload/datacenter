package handler

import (
	"github.com/gin-gonic/gin"

	"datacenter/internal/response"
	"datacenter/internal/service"
)

// AuditHandler 审计日志接口（admin 专属；系统优化 3.1）
type AuditHandler struct {
	svc *service.AuditService
}

// NewAuditHandler 构造审计日志处理器
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List GET /api/v1/audit-logs?action=&username=&page=&page_size=
func (h *AuditHandler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	logs, total, err := h.svc.List(c.Request.Context(),
		c.Query("action"), c.Query("username"), page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(logs, total, page, pageSize))
}

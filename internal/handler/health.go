package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"datacenter/internal/database"
	"datacenter/internal/errno"
	"datacenter/internal/response"
	"datacenter/internal/version"
)

// HealthHandler 健康检查
type HealthHandler struct {
	db *database.DB
}

// NewHealthHandler 构造健康检查处理器
func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health GET /healthz
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Client.Ping(ctx, nil); err != nil {
		response.Error(c, errno.ErrInternal.WithCause(err))
		return
	}
	response.OK(c, gin.H{
		"status":  "ok",
		"version": version.Version,
		"time":    time.Now().Format(time.RFC3339),
	})
}

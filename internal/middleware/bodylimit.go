package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit 请求体大小限制（安全增强 P0-4：防超大请求体 DoS）。
// 超出限制时读取方（gin binding）返回 *http.MaxBytesError，由 handler 的
// bindJSON 辅助统一映射为 413。
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

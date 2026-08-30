package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders 安全响应头（安全增强 P1-5 基础三项，零风险）：
//   - X-Content-Type-Options: nosniff  防 MIME 嗅探
//   - X-Frame-Options: DENY            防点击劫持（本系统无 iframe 场景）
//   - Referrer-Policy: same-origin     限制 Referer 泄露
//
// CSP 因前端存在内联 style 属性，列为实验项后置（见 安全增强方案.md §4.5）。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "same-origin")
		c.Next()
	}
}

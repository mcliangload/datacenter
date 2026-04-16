package logger

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

// ResponseWriter 自定义响应写入器，用于捕获响应内容
type ResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 重写Write方法，同时将响应内容写入缓冲区
func (w ResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggerMiddleware Gin日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 捕获请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 捕获响应体
		responseWriter := &ResponseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = responseWriter

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 获取用户身份（从JWT中获取）
		userID := c.GetString("userID")
		if userID == "" {
			userID = "anonymous"
		}

		// 构建日志数据
		logData := map[string]interface{}{
			"user_id":    userID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"query":      c.Request.URL.RawQuery,
			"status":     c.Writer.Status(),
			"latency":    latency.String(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		// 记录请求和响应内容（仅在debug级别）
		if len(requestBody) > 0 {
			logData["request_body"] = string(requestBody)
		}
		if responseWriter.body.Len() > 0 {
			logData["response_body"] = responseWriter.body.String()
		}

		// 根据状态码记录不同级别的日志
		status := c.Writer.Status()
		switch {
		case status >= 500:
			ErrorJSON(logData)
		case status >= 400:
			WarnJSON(logData)
		default:
			InfoJSON(logData)
		}
	}
}

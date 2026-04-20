package logger

import (
	"bytes"
	"time"

	"github.com/gin-gonic/gin"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// 创建响应体捕获器
		body := &bytes.Buffer{}
		w := &responseWriter{ResponseWriter: c.Writer, body: body}
		c.Writer = w

		c.Next()

		endTime := time.Now()
		latency := endTime.Sub(startTime)

		userID := c.GetString("userID")
		if userID == "" {
			userID = "anonymous"
		}

		status := c.Writer.Status()

		logEntry := HTTPLogger.With().
			Timestamp().
			Str("user_id", userID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("query", c.Request.URL.RawQuery).
			Int("status", status).
			Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Str("response", body.String()).
			Logger()

		switch {
		case status >= 500:
			logEntry.Error().Msg("")
		case status >= 400:
			logEntry.Warn().Msg("")
		default:
			logEntry.Info().Msg("")
		}
	}
}

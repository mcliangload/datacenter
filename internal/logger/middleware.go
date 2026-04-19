package logger

import (
	"time"

	"github.com/gin-gonic/gin"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

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

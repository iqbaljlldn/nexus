package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func Middleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.NewString()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		requestLogger := log.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
		)

		c.Request = c.Request.WithContext(IntoContext(c.Request.Context(), requestLogger))
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		requestLogger.Info("HTTP Request Completed",
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

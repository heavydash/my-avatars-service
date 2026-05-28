package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"time"
)

// StructuredLogger — middleware для Gin, который использует наш slog
func StructuredLogger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Запоминаем время начала
		start := time.Now()

		// Продолжаем обработку
		c.Next()

		// После обработки запроса логируем
		latency := time.Since(start)

		log.InfoCtx(c.Request.Context(),
			"HTTP Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"error", c.Errors.String(),
		)

	}
}

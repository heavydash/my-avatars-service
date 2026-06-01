package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/pkg/metrics"
	"strconv"
	"time"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Продолжаем обработку
		c.Next()

		// После обработки
		latency := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		path := c.FullPath()

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPLatency.WithLabelValues(method, path).Observe(latency)

		if status[0] == '4' || status[0] == '5' {
			metrics.HTTPErrorsTotal.WithLabelValues(method, path, status).Inc()
		}
	}
}

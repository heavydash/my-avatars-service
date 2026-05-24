package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

type RateLimiter struct {
	visits map[string][]time.Time
	mu     sync.RWMutex
	limit  int
	window time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visits: make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Очищаем старые записи
		valid := make([]time.Time, 0, rl.limit)
		for _, t := range rl.visits[ip] {
			if now.Sub(t) < rl.window {
				valid = append(valid, t)
			}
		}

		if len(valid) >= rl.limit {
			retryAfter := rl.window - now.Sub(valid[0])
			c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))

			c.JSON(429, gin.H{
				"error":       "too many requests",
				"retry_after": retryAfter.Seconds(),
			})
			c.Abort()
			return
		}

		valid = append(valid, now)
		rl.visits[ip] = valid

		c.Next()
	}
}

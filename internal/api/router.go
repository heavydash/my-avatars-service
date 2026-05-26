package api

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/api/handler"
	"github.com/heavydash/my-avatars-service/internal/api/middleware"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"time"
)

// NewRouter создаёт и настраивает Gin роутер
func NewRouter(avatarHandler *handler.AvatarHandler, log logger.Logger) *gin.Engine {
	r := gin.New()

	// Глобальные Middleware
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))

	// Rate Limiting: 10 запросов в минуту на IP
	rateLimiter := middleware.NewRateLimiter(10, 1*time.Minute)
	r.Use(rateLimiter.RateLimit())

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, X-User-ID")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	})

	// Публичные роуты
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "gophprofile",
		})
	})

	r.GET("/", func(c *gin.Context) {
		c.String(200, "GophProfile Avatar Service is running\n")
	})

	// Веб-интерфейс
	r.GET("/web/upload", func(c *gin.Context) {
		c.File("web/static/index.html")
	})
	r.Static("/web/static", "./web/static")

	// API v1
	v1 := r.Group("/api/v1")
	{
		avatars := v1.Group("/avatars")
		{
			avatars.POST("", avatarHandler.UploadAvatar)
			avatars.GET("/:id", avatarHandler.GetAvatar)
			avatars.GET("/:id/metadata", avatarHandler.GetAvatarMetadata)
			avatars.GET("", avatarHandler.GetUserAvatars)
			avatars.DELETE("/:id", avatarHandler.DeleteAvatar)
		}
		v1.GET("/users/:user_id/avatar", avatarHandler.GetUserAvatar)

	}

	return r
}

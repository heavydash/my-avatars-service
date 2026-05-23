package api

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/api/handler"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
)

// NewRouter создаёт и настраивает Gin роутер
func NewRouter(avatarHandler *handler.AvatarHandler, log logger.Logger) *gin.Engine {
	r := gin.New()

	// Глобальные Middleware
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))

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

	// API v1
	v1 := r.Group("/api/v1")
	{
		avatars := v1.Group("/avatars")
		{
			avatars.POST("", avatarHandler.UploadAvatar)
			avatars.GET("/:id", avatarHandler.GetAvatar)
			avatars.GET("", avatarHandler.GetUserAvatars)
			avatars.DELETE("/:id", avatarHandler.DeleteAvatar)
		}
	}

	return r
}

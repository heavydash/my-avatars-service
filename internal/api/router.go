package api

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/api/handler"
	"github.com/heavydash/my-avatars-service/internal/api/middleware"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/heavydash/my-avatars-service/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	_ "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"time"
)

// NewRouter создаёт и настраивает Gin роутер
func NewRouter(
	avatarHandler *handler.AvatarHandler,
	authHandler *handler.AuthHandler,
	jwtService *service.JWTService,
	log logger.Logger) *gin.Engine {
	r := gin.New()

	// OpenTelemetry Tracing Middleware
	r.Use(otelgin.Middleware("gophprofile"))

	// Глобальные Middleware
	r.Use(gin.Recovery())

	r.Use(middleware.PrometheusMiddleware())

	// Кастомный structured logger
	r.Use(middleware.StructuredLogger(log))

	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))

	// Rate Limiting: 10 запросов в минуту на IP
	rateLimiter := middleware.NewRateLimiter(10, 1*time.Minute)
	r.Use(rateLimiter.RateLimit())

	// Security Headers
	r.Use(securityHeadersMiddleware())

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
	public := r.Group("/")
	{
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "gophprofile",
			})
		})

		public.GET("/", func(c *gin.Context) {
			c.String(200, "GophProfile Avatar Service is running\n")
		})

		public.GET("/auth/test-token", authHandler.TestToken)

		// Веб-интерфейс
		r.GET("/web/upload", func(c *gin.Context) {
			c.File("web/static/index.html")
		})
		r.Static("/web/static", "./web/static")

		// API v1
		v1 := r.Group("/api/v1")
		// Публичные API роуты
		{
			v1.GET("/avatars/:id", avatarHandler.GetAvatar)
			v1.GET("/avatars/:id/metadata", avatarHandler.GetAvatarMetadata)
		}

		// Защищённые роуты
		authorized := v1.Group("/")
		authorized.Use(middleware.JWTAuth(jwtService))
		{
			avatars := authorized.Group("/avatars")
			{
				avatars.POST("", avatarHandler.UploadAvatar)
				avatars.GET("", avatarHandler.GetUserAvatars)
				avatars.DELETE("/:id", avatarHandler.DeleteAvatar)
			}
			v1.GET("/users/:user_id/avatar", avatarHandler.GetUserAvatar)

		}

		return r
	}
}

// Маленькие хендлеры
func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok", "service": "gophprofile"})
}

func rootHandler(c *gin.Context) {
	c.String(200, "GophProfile Avatar Service is running\n")
}

// securityHeadersMiddleware — middleware для security headers
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Access-Control-Allow-Origin", "*") // на проде лучше ограничить
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Next()
	}
}

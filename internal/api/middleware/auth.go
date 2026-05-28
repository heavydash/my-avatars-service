package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/service"
	"net/http"
	"strings"
)

// JWTAuth — middleware для Gin
func JWTAuth(jwtService *service.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token format. Use: Bearer <token>",
			})
			return
		}

		claims, err := jwtService.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		// Добавляем в контекст Gin
		c.Set("user_id", claims.UserID)
		c.Set("claims", claims)

		c.Next()
	}
}

// Хелперы для удобного получения из контекста
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

func GetClaims(c *gin.Context) (*service.Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	cl, ok := claims.(*service.Claims)
	return cl, ok
}

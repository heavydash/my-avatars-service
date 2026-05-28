package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/service"
	"net/http"
)

type AuthHandler struct {
	jwtService *service.JWTService
}

func NewAuthHandler(jwtService *service.JWTService) *AuthHandler {
	return &AuthHandler{
		jwtService: jwtService,
	}
}

// TestToken — временный эндпоинт только для разработки
func (h *AuthHandler) TestToken(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "550e8400-e29b-41d4-a716-446655440000"
	}

	token, err := h.jwtService.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"user_id":      userID,
		"expires_in":   "24h",
	})
}

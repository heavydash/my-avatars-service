package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/heavydash/my-avatars-service/internal/service"
	"net/http"
)

type AuthHandler struct {
	jwtService *service.JWTService
	log        logger.Logger
}

func NewAuthHandler(jwtService *service.JWTService, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		jwtService: jwtService,
		log:        log,
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
		h.log.Error("Failed to generate JWT token",
			"user_id", userID,
			"error", err,
		)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"user_id":      userID,
		"expires_in":   "24h",
	})
}

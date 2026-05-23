package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/heavydash/my-avatars-service/internal/service"
	"net/http"
)

type AvatarHandler struct {
	service *service.AvatarService
}

func NewAvatarHandler(svc *service.AvatarService) *AvatarHandler {
	return &AvatarHandler{service: svc}
}

// UploadAvatar — загрузка аватарки
func (h *AvatarHandler) UploadAvatar(c *gin.Context) {
	userIDStr := c.PostForm("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}
	defer src.Close()

	avatar, err := h.service.UploadAvatar(c.Request.Context(), userID, src, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, avatar)
}

// GetAvatar — получение одной аватарки
func (h *AvatarHandler) GetAvatar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar id"})
		return
	}

	avatar, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusOK, avatar)
}

// GetUserAvatars — получение аватарок пользователя
func (h *AvatarHandler) GetUserAvatars(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	avatars, err := h.service.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusOK, avatars)
}

// DeleteAvatar — удаление аватарки
func (h *AvatarHandler) DeleteAvatar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar id"})
		return
	}

	if err := h.service.DeleteAvatar(c.Request.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

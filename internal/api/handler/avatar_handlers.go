package handler

import (
	"fmt"
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

	size := c.DefaultQuery("size", "original")
	format := c.DefaultQuery("format", "")

	avatar, err := h.service.GetByIDWithOptions(c.Request.Context(), id, size, format)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}

	// TODO: здесь будет логика выбора thumbnail по size/format

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

	// Получаем последнюю аватарку пользователя
	avatars, err := h.service.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusOK, avatars)
}

// GetUserAvatar — эндпоинт последней аватарки пользователя
func (h *AvatarHandler) GetUserAvatar(c *gin.Context) {
	userIDStr := c.Param("user_id")
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

	if len(avatars) == 0 {
		// Заглушка — можно позже заменить на реальный default avatar
		c.Redirect(http.StatusTemporaryRedirect, "/web/default-avatar.png")
		return
	}

	// Возвращаем самую свежую аватарку
	latest := avatars[0]
	c.Redirect(http.StatusTemporaryRedirect, latest.OriginalURL)
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

// GetAvatarMetadata — получение метаданных аватарки
func (h *AvatarHandler) GetAvatarMetadata(c *gin.Context) {
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

	// Формируем метаданные
	c.JSON(http.StatusOK, gin.H{
		"id":           avatar.ID,
		"user_id":      avatar.UserID,
		"original_url": avatar.OriginalURL,
		"file_size":    avatar.FileSize,
		"content_type": avatar.ContentType,
		"status":       avatar.Status,
		"created_at":   avatar.CreatedAt,
		"updated_at":   avatar.UpdatedAt,
		"thumbnails": []gin.H{
			{
				"size": "100x100",
				"url":  fmt.Sprintf("http://localhost:9000/avatars/thumbnails/%s/100x100.jpg", avatar.ID),
			},
			{
				"size": "300x300",
				"url":  fmt.Sprintf("http://localhost:9000/avatars/thumbnails/%s/300x300.jpg", avatar.ID),
			},
		},
	})
}

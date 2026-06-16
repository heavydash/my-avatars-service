package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/api/middleware"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/heavydash/my-avatars-service/internal/service"
	"log"
	"net/http"
)

type AvatarHandler struct {
	jwtService service.JWTService
	service    service.AvatarUseCase
}

func NewAvatarHandler(svc service.AvatarUseCase, jwtService service.JWTService) *AvatarHandler {
	return &AvatarHandler{
		service:    svc,
		jwtService: jwtService,
	}
}

// UploadAvatar загружает аватарку пользователя
// @Summary      Загрузить аватарку
// @Description  Загружает файл аватарки (jpg, png, webp). Требуется авторизация.
// @Tags         avatars
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Файл аватарки"
// @Success      201   {object}  domain.Avatar
// @Failure      400   {object}  map[string]string  "Неверный файл или формат"
// @Failure      401   {object}  map[string]string  "Не авторизован"
// @Failure      413   {object}  map[string]string  "Файл слишком большой"
// @Router       /api/v1/avatars [post]
func (h *AvatarHandler) UploadAvatar(c *gin.Context) {
	// Берём user_id из JWT
	userIDStr, exists := middleware.GetUserID(c)
	if !exists || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id is required (from JWT)"})
		return
	}

	// Парсим string → uuid.UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id in token"})
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

// GetAvatar отдаёт аватарку (редирект на файл)
// @Summary      Получить аватарку
// @Description  Возвращает редирект на оригинал или миниатюру аватарки
// @Tags         avatars
// @Produce      json
// @Param        id     path      string  true  "ID аватарки"
// @Param        size   query     string  false "Размер миниатюры (100x100, 300x300)"
// @Param        format query     string  false "Формат (jpg, webp)"
// @Success      307   "Редирект на файл"
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string  "Аватарка не найдена"
// @Router       /api/v1/avatars/{id} [get]
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

	// Логика выбора URL
	url := avatar.OriginalURL
	if size != "original" {
		url = fmt.Sprintf("http://localhost:9000/avatars/thumbnails/%s/%s.jpg", avatar.ID, size)
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GetUserAvatars возвращает все аватарки пользователя
// @Summary      Получить все аватарки пользователя
// @Tags         avatars
// @Produce      json
// @Param        user_id  query  string  true  "ID пользователя"
// @Success      200  {array}   domain.Avatar
// @Failure      400  {object}  map[string]string
// @Router       /api/v1/avatars [get]
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
		log.Printf("GetByUserID error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}

	c.JSON(http.StatusOK, avatars)
}

// GetUserAvatar отдаёт последнюю аватарку пользователя (редирект)
// @Summary      Получить последнюю аватарку пользователя
// @Tags         avatars
// @Produce      json
// @Param        user_id  path  string  true  "ID пользователя"
// @Success      307  "Редирект на аватарку"
// @Failure      400  {object}  map[string]string
// @Router       /api/v1/users/{user_id}/avatar [get]
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
		c.Redirect(http.StatusTemporaryRedirect, "/web/default-avatar.jpg")
		return
	}

	// Возвращаем самую свежую аватарку
	latest := avatars[0]
	c.Redirect(http.StatusTemporaryRedirect, latest.OriginalURL)
}

// DeleteAvatar удаляет аватарку
// @Summary      Удалить аватарку
// @Description  Удаляет аватарку. Можно удалять только свои аватарки.
// @Tags         avatars
// @Produce      json
// @Param        id          path   string  true  "ID аватарки"
// @Param        X-User-ID   header string  true  "ID пользователя"
// @Success      204  "Аватарка удалена"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string  "Нет прав на удаление"
// @Failure      404  {object}  map[string]string  "Аватарка не найдена"
// @Router       /api/v1/avatars/{id} [delete]
func (h *AvatarHandler) DeleteAvatar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid avatar id"})
		return
	}

	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-ID header is required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-User-ID"})
		return
	}

	// Получаем аватарку, чтобы проверить владельца
	avatar, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrInternal.Error()})
		return
	}

	// Проверка владения
	if avatar.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own avatars"})
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

// GetAvatarMetadata возвращает метаданные аватарки
// @Summary      Получить метаданные аватарки
// @Tags         avatars
// @Produce      json
// @Param        id  path  string  true  "ID аватарки"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /api/v1/avatars/{id}/metadata [get]
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

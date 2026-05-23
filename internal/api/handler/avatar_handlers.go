package handler

import (
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/service"
	"github.com/labstack/echo/v4"
	"net/http"
)

type AvatarHandler struct {
	service *service.AvatarService
}

func NewAvatarHandler(svc *service.AvatarService) *AvatarHandler {
	return &AvatarHandler{service: svc}
}

// UploadAvatar — обработка загрузки аватарки
func (h *AvatarHandler) UploadAvatar(c echo.Context) error {
	userIDStr := c.FormValue("user_id")
	if userIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user_id format"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open file"})
	}
	defer src.Close()

	avatar, err := h.service.UploadAvatar(c.Request().Context(), userID, src, file)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, avatar)
}

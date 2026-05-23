package domain

import (
	"github.com/google/uuid"
	"time"
)

// AvatarStatus — статус обработки аватарки
type AvatarStatus string

const (
	AvatarStatusUploading AvatarStatus = "uploading"
	AvatarStatusReady     AvatarStatus = "ready"
	AvatarStatusFailed    AvatarStatus = "failed"
)

// Avatar — основная доменная сущность
type Avatar struct {
	ID          uuid.UUID    `json:"id"`
	UserID      uuid.UUID    `json:"user_id"`
	OriginalURL string       `json:"original_url"`
	FileSize    int64        `json:"file_size"`
	ContentType string       `json:"content_type"`
	Status      AvatarStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Error       string       `json:"error,omitempty"`
}

// NewAvatar создаёт новую аватарку в статусе uploading
func NewAvatar(userID uuid.UUID, originalURL string, fileSize int64, contentType string) *Avatar {
	now := time.Now().UTC()
	return &Avatar{
		ID:          uuid.New(),
		UserID:      userID,
		OriginalURL: originalURL,
		FileSize:    fileSize,
		ContentType: contentType,
		Status:      AvatarStatusUploading,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MarkAsReady переводит аватарку в статус ready
func (a *Avatar) MarkAsReady() {
	a.Status = AvatarStatusReady
	a.UpdatedAt = time.Now().UTC()
}

// MarkAsFailed переводит аватарку в статус failed
func (a *Avatar) MarkAsFailed(err string) {
	a.Status = AvatarStatusFailed
	a.Error = err
	a.UpdatedAt = time.Now().UTC()
}

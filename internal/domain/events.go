package domain

// AvatarUploadedEvent — событие после загрузки аватарки
type AvatarUploadedEvent struct {
	AvatarID    string `json:"avatar_id"`
	UserID      string `json:"user_id"`
	OriginalURL string `json:"original_url"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
}

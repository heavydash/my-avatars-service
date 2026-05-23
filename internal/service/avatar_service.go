package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/heavydash/my-avatars-service/internal/repository"
	"github.com/heavydash/my-avatars-service/internal/storage"
	"mime/multipart"
)

// AvatarService — бизнес-логика работы с аватарками
type AvatarService struct {
	repo    repository.AvatarRepository
	storage storage.Storage
}

// NewAvatarService сервис добавления аватарок
func NewAvatarService(repo repository.AvatarRepository, storage storage.Storage) *AvatarService {
	return &AvatarService{
		repo:    repo,
		storage: storage,
	}
}

// UploadAvatar обрабатывает загрузку аватарки
func (s *AvatarService) UploadAvatar(ctx context.Context, userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Avatar, error) {
	// Валидация размера
	if header.Size > 10*1024*1024 { // 10MB
		return nil, fmt.Errorf("file size exceeds 10MB limit")
	}

	// Валидация типа файла
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		return nil, fmt.Errorf("unsupported file type: %s", contentType)
	}

	// Создаём доменную сущность
	avatar := domain.NewAvatar(userID, "", header.Size, contentType)

	// Сохраняем файл в MinIO
	url, err := s.storage.Save(ctx, avatar.ID.String(), file, header)
	if err != nil {
		avatar.MarkAsFailed(err.Error())
		_ = s.repo.Create(ctx, avatar)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	avatar.OriginalURL = url
	avatar.MarkAsReady()

	// Сохраняем метаданные в БД
	if err := s.repo.Create(ctx, avatar); err != nil {
		return nil, fmt.Errorf("failed to save avatar metadata: %w", err)
	}

	return avatar, nil
}

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
	// Валидация user_id
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	// Валидация размера
	if header.Size > 10*1024*1024 { // 10MB
		return nil, domain.ErrFileTooLarge
	}

	// Разрешённые типы файлов
	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"image/gif":       true,
		"application/pdf": true,
	}

	if !allowedTypes[contentType] {
		return nil, domain.ErrUnsupportedFormat
	}

	// Создаём доменную сущность
	avatar := domain.NewAvatar(userID, "", header.Size, contentType)

	// Сохраняем файл в MinIO
	url, err := s.storage.Save(ctx, avatar.ID.String(), file, header)
	if err != nil {
		avatar.MarkAsFailed(err.Error())
		_ = s.repo.Create(ctx, avatar)
		return nil, fmt.Errorf("%w: %w", domain.ErrUploadFailed, err)
	}

	avatar.OriginalURL = url
	avatar.MarkAsReady()

	// Сохраняем метаданные в БД
	if err := s.repo.Create(ctx, avatar); err != nil {
		return nil, domain.ErrInternal
	}

	return avatar, nil
}

// DeleteAvatar — удаление аватарки
func (s *AvatarService) DeleteAvatar(ctx context.Context, id uuid.UUID) error {
	avatar, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Удаляем файл из MinIO
	if err := s.storage.Delete(ctx, avatar.ID.String()); err != nil {
		fmt.Printf("Warning: failed to delete file from storage: %v\n", err)
	}

	// Удаляем метаданные из БД
	return s.repo.Delete(ctx, id)
}

// GetByID — получение аватарки по ID
func (s *AvatarService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByUserID — получение всех аватарок пользователя
func (s *AvatarService) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error) {
	return s.repo.GetByUserID(ctx, userID)
}

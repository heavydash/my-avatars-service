package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/heavydash/my-avatars-service/internal/events"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/heavydash/my-avatars-service/internal/repository"
	"github.com/heavydash/my-avatars-service/internal/storage"
	"go.uber.org/zap"
	"mime/multipart"
)

// AvatarService — бизнес-логика работы с аватарками
type AvatarService struct {
	repo      repository.AvatarRepository
	storage   storage.Storage
	publisher *events.Publisher
	logger    logger.Logger
}

// NewAvatarService сервис добавления аватарок
func NewAvatarService(repo repository.AvatarRepository, storage storage.Storage, publisher *events.Publisher, log logger.Logger) *AvatarService {
	return &AvatarService{
		repo:      repo,
		storage:   storage,
		publisher: publisher,
		logger:    log,
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

	// Magic bytes валидация
	if err := validateFileType(file, contentType); err != nil {
		return nil, err
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

	// Публикуем событие для асинхронной обработки
	if s.publisher != nil {
		event := domain.AvatarUploadedEvent{
			AvatarID:    avatar.ID.String(),
			UserID:      avatar.UserID.String(),
			OriginalURL: avatar.OriginalURL,
			FileSize:    avatar.FileSize,
			ContentType: avatar.ContentType,
		}

		if err := s.publisher.PublishAvatarUploaded(ctx, event); err != nil {
			// Пока только warning, не падаем — загрузка уже прошла успешно
			s.logger.Warn("failed to publish upload event: %v\n",
				zap.String("avatar_id", avatar.ID.String()),
				zap.Error(err))
		} else {
			s.logger.Info("Event published successfully",
				zap.String("avatar_id", avatar.ID.String()),
				zap.String("user_id", avatar.UserID.String()))
		}
	}

	return avatar, nil
}

// DeleteAvatar — удаление аватарки
func (s *AvatarService) DeleteAvatar(ctx context.Context, id uuid.UUID) error {
	// Получаем аватарку, чтобы проверить существование и получить ключ для MinIO
	avatar, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Публикуем событие для асинхронного удаления файлов из MinIO
	if s.publisher != nil {
		event := domain.AvatarDeleteEvent{
			AvatarID: avatar.ID.String(),
			S3Keys:   []string{avatar.ID.String()},
		}
		if err := s.publisher.PublishAvatarDeleted(ctx, event); err != nil {
			s.logger.Warn("Failed to publish delete event",
				zap.String("avatar_id", avatar.ID.String()),
				zap.Error(err))
		}
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

// GetByIDWithOptions — получение аватарки с параметрами размера и формата
func (s *AvatarService) GetByIDWithOptions(ctx context.Context, id uuid.UUID, size, format string) (*domain.Avatar, error) {
	return s.repo.GetByID(ctx, id)
}

// validateFileType — проверка magic bytes
func validateFileType(file multipart.File, declaredContentType string) error {
	// Читаем первые 512 байт
	header := make([]byte, 512)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrUnsupportedFormat, err)
	}

	// Возвращаем указатель обратно
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Проверяем реальный тип файла
	kind, err := filetype.Match(header)
	if err != nil {
		return domain.ErrUnsupportedFormat
	}

	if kind == filetype.Unknown {
		return domain.ErrUnsupportedFormat
	}

	// Разрешённые реальные типы
	allowed := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"image/gif":       true,
		"application/pdf": true,
	}

	if !allowed[kind.MIME.Value] {
		return domain.ErrUnsupportedFormat
	}

	return nil
}

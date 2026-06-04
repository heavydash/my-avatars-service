package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/heavydash/my-avatars-service/internal/events"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/heavydash/my-avatars-service/internal/pkg/metrics"
	"github.com/heavydash/my-avatars-service/internal/repository"
	"github.com/heavydash/my-avatars-service/internal/storage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"mime/multipart"
	"time"
)

// AvatarService — бизнес-логика работы с аватарками
type AvatarService struct {
	repo      repository.AvatarRepository
	storage   storage.Storage
	publisher events.PublisherInterface
	logger    logger.Logger
	tracer    trace.Tracer
}

// NewAvatarService сервис добавления аватарок
func NewAvatarService(repo repository.AvatarRepository, storage storage.Storage, publisher events.PublisherInterface, log logger.Logger) *AvatarService {
	return &AvatarService{
		repo:      repo,
		storage:   storage,
		publisher: publisher,
		logger:    log,
		tracer:    otel.Tracer("gophprofile.service"),
	}
}

// UploadAvatar обрабатывает загрузку аватарки
func (s *AvatarService) UploadAvatar(ctx context.Context, userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Avatar, error) {
	start := time.Now()

	ctx, span := s.tracer.Start(ctx, "AvatarService.UploadAvatar", trace.WithAttributes(
		attribute.String("user_id", userID.String()),
		attribute.Int64("file_size", header.Size)))
	defer span.End()

	// Валидация user_id
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	// Валидация размера
	if header.Size > 10*1024*1024 { // 10MB
		s.logger.WarnCtx(ctx, "File too large", "size", header.Size, "user_id", userID)
		return nil, domain.ErrFileTooLarge
	}

	// Разрешённые типы файлов
	contentType := header.Header.Get("Content-Type")
	// Magic bytes валидация
	if err := validateFileType(file, contentType); err != nil {
		s.logger.WarnCtx(ctx, "Unsupported file type", "content_type", contentType, "user_id", userID)
		return nil, err
	}

	// Создаём доменную сущность
	avatar := domain.NewAvatar(userID, "", header.Size, contentType)

	s.logger.InfoCtx(ctx, "Starting avatar upload",
		"user_id", userID,
		"file_size", header.Size,
		"content_type", contentType,
	)

	// Сохраняем файл в MinIO
	url, err := s.storage.Save(ctx, avatar.ID.String(), file, header)
	if err != nil {
		avatar.MarkAsFailed(err.Error())
		_ = s.repo.Create(ctx, avatar)
		s.logger.ErrorCtx(ctx, "Failed to save file to storage", "error", err, "user_id", userID)
		return nil, fmt.Errorf("%w: %w", domain.ErrUploadFailed, err)
	}

	avatar.OriginalURL = url
	avatar.MarkAsReady()

	// Сохраняем метаданные в БД
	if err := s.repo.Create(ctx, avatar); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to save avatar metadata", "error", err, "user_id", userID)
		metrics.AvatarUploadsTotal.WithLabelValues("failed").Inc()
		return nil, domain.ErrInternal
	}

	// Успешная загрузка
	duration := time.Since(start).Seconds()
	metrics.AvatarUploadDuration.Observe(duration)
	metrics.AvatarUploadsTotal.WithLabelValues("success").Inc()
	metrics.StorageUsageBytes.WithLabelValues(userID.String()).Add(float64(header.Size))
	metrics.StorageObjectsTotal.Inc()

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
			s.logger.WarnCtx(ctx, "Failed to publish upload event", "error", err, "avatar_id", avatar.ID)
		} else {
			s.logger.InfoCtx(ctx, "Avatar uploaded and event published",
				"avatar_id", avatar.ID,
				"user_id", userID,
			)
		}
	}

	return avatar, nil
}

// DeleteAvatar — удаление аватарки
func (s *AvatarService) DeleteAvatar(ctx context.Context, id uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "AvatarService.DeleteAvatar", trace.WithAttributes(
		attribute.String("avatar_id", id.String())))
	defer span.End()

	// Получаем аватарку, чтобы проверить существование и получить ключ для MinIO
	avatar, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			metrics.AvatarDeletesTotal.WithLabelValues("not_found").Inc()
		} else {
			metrics.AvatarDeletesTotal.WithLabelValues("failed").Inc()
		}
		return err
	}

	s.logger.InfoCtx(ctx, "Starting avatar deletion",
		"avatar_id", avatar.ID,
		"user_id", avatar.UserID,
	)

	// Публикуем событие для асинхронного удаления файлов из MinIO
	if s.publisher != nil {
		event := domain.AvatarDeleteEvent{
			AvatarID: avatar.ID.String(),
			S3Keys:   []string{avatar.ID.String()},
		}
		if err := s.publisher.PublishAvatarDeleted(ctx, event); err != nil {
			s.logger.Warn("Failed to publish delete event",
				slog.String("avatar_id", avatar.ID.String()),
				"error", err)
		}
	}

	// Удаляем метаданные из БД
	if err := s.repo.Delete(ctx, id); err != nil {
		metrics.AvatarDeletesTotal.WithLabelValues("failed").Inc()
		s.logger.ErrorCtx(ctx, "Failed to delete avatar from database", "error", err, "avatar_id", avatar.ID)
		return err
	}

	metrics.AvatarDeletesTotal.WithLabelValues("success").Inc()
	metrics.StorageUsageBytes.WithLabelValues(avatar.UserID.String()).Sub(float64(avatar.FileSize))
	metrics.StorageObjectsTotal.Dec()

	s.logger.InfoCtx(ctx, "Avatar deleted successfully",
		"avatar_id", avatar.ID,
		"user_id", avatar.UserID,
	)

	return nil
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

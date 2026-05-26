package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/heavydash/my-avatars-service/internal/repository"
	"github.com/heavydash/my-avatars-service/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"image"
	"time"
)

type Worker struct {
	dbPool  *pgxpool.Pool
	channel *amqp091.Channel
	logger  logger.Logger
	repo    repository.AvatarRepository
	minio   storage.Storage
}

func NewWorker(dbPool *pgxpool.Pool, ch *amqp091.Channel, log logger.Logger, repo repository.AvatarRepository, minio storage.Storage) *Worker {
	return &Worker{
		dbPool:  dbPool,
		channel: ch,
		logger:  log,
		repo:    repo,
		minio:   minio,
	}
}

func (w *Worker) Start() error {
	w.logger.Info("Worker started, waiting for messages...")

	// Объявляем очередь
	_, err := w.channel.QueueDeclare(
		"avatar.processing", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err
	}

	// Биндим очередь к exchange
	err = w.channel.QueueBind(
		"avatar.processing",
		"avatar.uploaded",
		"avatars.exchange",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := w.channel.Consume(
		"avatar.processing", // queue
		"",                  // consumer
		false,               // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	if err != nil {
		return err
	}

	for msg := range msgs {
		w.logger.Info("Received message", zap.String("body", string(msg.Body)))

		var event domain.AvatarUploadedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			w.logger.Error("Failed to unmarshal event", zap.Error(err))
			msg.Nack(false, false) // не возвращаем в очередь
			continue
		}

		w.logger.Info("Processing avatar",
			zap.String("avatar_id", event.AvatarID),
			zap.String("user_id", event.UserID))

		avatarID, err := uuid.Parse(event.AvatarID)
		if err != nil {
			w.logger.Error("Invalid avatar ID format in event",
				zap.String("avatar_id", event.AvatarID),
				zap.Error(err))
			msg.Nack(false, false) // битое сообщение — не возвращаем
			continue
		}

		avatar, err := w.repo.GetByID(context.Background(), avatarID)
		if err != nil {
			if err == domain.ErrNotFound {
				w.logger.Warn("avatar not found",
					zap.String("avatar_id", event.AvatarID))
				msg.Ack(false)
				continue
			}
			w.logger.Error("Failed to get avatar", zap.Error(err))
			msg.Nack(false, true)
			continue
		}

		if avatar.Status == domain.AvatarStatusReady {
			w.logger.Info("avatar already processed, skipping",
				zap.String("avatar_id", event.AvatarID))
			msg.Ack(false)
			continue
		}

		// Здесь будет основная обработка
		if err := w.processImage(event, avatarID); err != nil {
			w.logger.Error("Failed to process image", zap.Error(err))
			msg.Nack(false, true) // возвращаем в очередь для retry
			continue
		}

		msg.Ack(false) // успешно отработано
	}

	return nil
}

// processImage — основная обработка изображения
func (w *Worker) processImage(event domain.AvatarUploadedEvent, avatarID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Пропускаем обработку для не-изображений (например, PDF)
	if !isImage(event.ContentType) {
		w.logger.Info("Skipping image processing for non-image file",
			zap.String("content_type", event.ContentType),
			zap.String("avatar_id", event.AvatarID))

		return w.updateAvatarStatus(ctx, avatarID, domain.AvatarStatusReady)
	}

	file, err := w.minio.GetObject(ctx, event.AvatarID)
	if err != nil {
		return fmt.Errorf("failed to download original: %w", err)
	}
	defer file.Close()

	w.logger.Debug("Decoding image...")
	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}
	w.logger.Debug("Image decoded successfully")

	// Создаём версии
	sizes := map[string]image.Point{
		"100x100": {100, 100},
		"300x300": {300, 300},
	}

	for sizeName, size := range sizes {
		w.logger.Debug("Creating thumbnail", zap.String("size", sizeName))
		thumb := imaging.Resize(img, size.X, size.Y, imaging.Lanczos)

		var buf bytes.Buffer
		if err = imaging.Encode(&buf, thumb, imaging.JPEG); err != nil {
			w.logger.Warn("Failed to encode thumbnail", zap.String("size", sizeName), zap.Error(err))
			continue
		}

		thumbKey := fmt.Sprintf("thumbnails/%s/%s.jpg", event.AvatarID, sizeName)

		if _, err = w.minio.SaveFromBytes(ctx, thumbKey, buf.Bytes(), "image/jpeg"); err != nil {
			w.logger.Warn("Failed to save thumbnail",
				zap.String("size", sizeName), zap.Error(err))
		} else {
			w.logger.Info("Thumbnail saved", zap.String("size", sizeName))
		}
	}

	// Обновляем статус
	if err := w.updateAvatarStatus(ctx, avatarID, domain.AvatarStatusReady); err != nil {
		w.logger.Warn("Failed to update avatar status", zap.Error(err))
	}

	w.logger.Info("Image processing completed successfully", zap.String("avatar_id", event.AvatarID))
	return nil
}

// isImage — проверяет, является ли файл изображением
func isImage(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

// updateAvatarStatus — вспомогательный метод
func (w *Worker) updateAvatarStatus(ctx context.Context, avatarID uuid.UUID, status domain.AvatarStatus) error {
	avatar, err := w.repo.GetByID(ctx, avatarID)
	if err != nil {
		return err
	}

	avatar.Status = status
	avatar.UpdatedAt = time.Now().UTC()

	return w.repo.Update(ctx, avatar)
}

func (w *Worker) Stop() {
	w.logger.Info("Stopping worker...")
}

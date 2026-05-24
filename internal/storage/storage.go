package storage

import (
	"context"
	"io"
	"mime/multipart"
)

// Storage — порт для работы с файловым хранилищем (MinIO / S3)
type Storage interface {
	// Save сохраняет файл и возвращает публичный URL
	Save(ctx context.Context, objectName string, file multipart.File, header *multipart.FileHeader) (string, error)

	// GetObject — скачивает файл (нужен для Worker)
	GetObject(ctx context.Context, objectName string) (io.ReadCloser, error)

	// SaveFromBytes — сохраняет байты (нужен для сохранения thumbnails)
	SaveFromBytes(ctx context.Context, objectName string, data []byte, contentType string) (string, error)

	// GetURL возвращает публичный URL файла
	GetURL(objectName string) string

	// Delete удаляет файл
	Delete(ctx context.Context, objectName string) error
}

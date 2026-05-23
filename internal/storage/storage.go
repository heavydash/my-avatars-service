package storage

import (
	"context"
	"mime/multipart"
)

// Storage — порт для работы с файловым хранилищем
type Storage interface {
	// Save сохраняет файл и возвращает публичный URL
	Save(ctx context.Context, objectName string, file multipart.File, header *multipart.FileHeader) (string, error)

	// GetURL возвращает URL файла
	GetURL(objectName string) string

	// Delete удаляет файл
	Delete(ctx context.Context, objectName string) error
}

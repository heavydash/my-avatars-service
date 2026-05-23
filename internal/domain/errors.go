package domain

import "errors"

var (
	// Общие ошибки
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal error")

	// Аватарки
	ErrFileTooLarge      = errors.New("file size exceeds 10MB limit")
	ErrUnsupportedFormat = errors.New("unsupported file type")
	ErrUploadFailed      = errors.New("failed to upload file")
)

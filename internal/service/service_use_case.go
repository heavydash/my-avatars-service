package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"mime/multipart"
)

type AvatarUseCase interface {
	UploadAvatar(ctx context.Context, userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Avatar, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error)
	GetByIDWithOptions(ctx context.Context, id uuid.UUID, size, format string) (*domain.Avatar, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error)
	DeleteAvatar(ctx context.Context, id uuid.UUID) error
}

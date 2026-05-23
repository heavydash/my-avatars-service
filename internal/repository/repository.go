package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
)

// AvatarRepository — порт для работы с хранилищем аватарок
type AvatarRepository interface {
	Create(ctx context.Context, avatar *domain.Avatar) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error)
	Update(ctx context.Context, avatar *domain.Avatar) error
	Delete(ctx context.Context, id uuid.UUID) error
}

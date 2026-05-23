package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AvatarRepository struct {
	db *pgxpool.Pool
}

func NewAvatarRepository(db *pgxpool.Pool) *AvatarRepository {
	return &AvatarRepository{db: db}
}

func (r *AvatarRepository) Create(ctx context.Context, avatar *domain.Avatar) error {
	query := `
		INSERT INTO avatars (id, user_id, original_url, file_size, content_type, status, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.Exec(ctx, query,
		avatar.ID,
		avatar.UserID,
		avatar.OriginalURL,
		avatar.FileSize,
		avatar.ContentType,
		avatar.Status,
		avatar.Error,
		avatar.CreatedAt,
		avatar.UpdatedAt,
	)
	return err
}

func (r *AvatarRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	// будет реализовано позже
	return nil, nil
}

func (r *AvatarRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error) {
	// будет реализовано позже
	return nil, nil
}

func (r *AvatarRepository) Update(ctx context.Context, avatar *domain.Avatar) error {
	// будет реализовано позже
	return nil
}

func (r *AvatarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// будет реализовано позже
	return nil
}

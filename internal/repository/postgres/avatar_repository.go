package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/jackc/pgx/v5"
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
	query := `
		SELECT id, user_id, original_url, file_size, content_type, status, error, created_at, updated_at
		FROM avatars 
		WHERE id = $1`

	var a domain.Avatar
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID,
		&a.UserID,
		&a.OriginalURL,
		&a.FileSize,
		&a.ContentType,
		&a.Status,
		&a.Error,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *AvatarRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error) {
	query := `
		SELECT id, user_id, original_url, file_size, content_type, status, error, created_at, updated_at
		FROM avatars 
		WHERE user_id = $1 
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var avatars []*domain.Avatar
	for rows.Next() {
		var a domain.Avatar
		err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.OriginalURL,
			&a.FileSize,
			&a.ContentType,
			&a.Status,
			&a.Error,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		avatars = append(avatars, &a)
	}
	return avatars, nil
}

func (r *AvatarRepository) Update(ctx context.Context, avatar *domain.Avatar) error {
	// будет реализовано позже
	return nil
}

func (r *AvatarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// будет реализовано позже
	return nil
}

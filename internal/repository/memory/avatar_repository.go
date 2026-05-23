package memory

import (
	"context"
	"github.com/google/uuid"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"sync"
)

type AvatarRepository struct {
	mu      sync.RWMutex
	avatars map[uuid.UUID]*domain.Avatar
}

func NewAvatarRepository() *AvatarRepository {
	return &AvatarRepository{
		avatars: make(map[uuid.UUID]*domain.Avatar),
	}
}

func (r *AvatarRepository) Create(ctx context.Context, avatar *domain.Avatar) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.avatars[avatar.ID] = avatar
	return nil
}

func (r *AvatarRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Avatar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	avatar, ok := r.avatars[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return avatar, nil
}
func (r *AvatarRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Avatar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Avatar
	for _, a := range r.avatars {
		if a.UserID == userID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (r *AvatarRepository) Update(ctx context.Context, avatar *domain.Avatar) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.avatars[avatar.ID] = avatar
	return nil
}

func (r *AvatarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.avatars, id)
	return nil
}

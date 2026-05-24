package events

import (
	"context"
	"github.com/heavydash/my-avatars-service/internal/domain"
)

// internal/events/publisher.go
type PublisherInterface interface {
	PublishAvatarUploaded(ctx context.Context, event domain.AvatarUploadedEvent) error
	PublishAvatarDeleted(ctx context.Context, event domain.AvatarDeleteEvent) error
}

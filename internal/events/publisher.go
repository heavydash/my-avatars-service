package events

import (
	"context"
	"encoding/json"
	"github.com/heavydash/my-avatars-service/internal/domain"
	"github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch *amqp091.Channel
}

func NewPublisher(ch *amqp091.Channel) *Publisher {
	return &Publisher{ch: ch}
}

func (p *Publisher) PublishAvatarUploaded(ctx context.Context, event domain.AvatarUploadedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(ctx,
		"avatars.exchange", // exchange
		"avatar.uploaded",  // routing key
		false,              // mandatory
		false,              // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// PublishAvatarDeleted — публикует событие удаления
func (p *Publisher) PublishAvatarDeleted(ctx context.Context, event domain.AvatarDeleteEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(ctx,
		"avatars.exchange",
		"avatar.deleted",
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

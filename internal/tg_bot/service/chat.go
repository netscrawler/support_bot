package service

import (
	"context"
	"fmt"
	"log/slog"

	models "support_bot/internal/models/notify"
)

type ChatProvider interface {
	Create(ctx context.Context, chat *models.Chat) error
	GetByTitle(ctx context.Context, title string) (*models.Chat, error)
	GetAll(ctx context.Context) ([]models.Chat, error)
	Delete(ctx context.Context, chatID int64) error
}

type Chat struct {
	repo ChatProvider
	log  *slog.Logger
}

func NewChat(repo ChatProvider, log *slog.Logger) *Chat {
	l := log.With(slog.Any("module", "tg_bot.service.chat"))
	return &Chat{
		repo: repo,
		log:  l,
	}
}

func (c *Chat) AddActive(ctx context.Context, chat *models.Chat) error {
	ch, _ := c.repo.GetByTitle(ctx, chat.Title)
	if ch != nil {
		return models.ErrAlreadyExist
	}

	err := c.repo.Create(ctx, chat)
	if err != nil {
		return fmt.Errorf("%w %w", models.ErrInternal, err)
	}

	return nil
}

func (c *Chat) Add(ctx context.Context, chat *models.Chat) error {
	ch, _ := c.repo.GetByTitle(ctx, chat.Title)
	if ch != nil {
		return models.ErrAlreadyExist
	}

	err := c.repo.Create(ctx, chat)
	if err != nil {
		return fmt.Errorf("%w %w", models.ErrInternal, err)
	}

	return nil
}

func (c *Chat) Remove(ctx context.Context, title string) error {
	ch, err := c.repo.GetByTitle(ctx, title)
	if err != nil {
		return err
	}

	chID := ch.ChatID

	return c.repo.Delete(ctx, chID)
}

func (c *Chat) GetAll(ctx context.Context) ([]models.Chat, error) {
	chats, err := c.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

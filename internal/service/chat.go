package service

import (
	"context"
	"fmt"
	"support_bot/internal/models"
)

type ChatProvider interface {
	Create(ctx context.Context, chat *models.Chat) error
	GetByTitle(ctx context.Context, title string) (*models.Chat, error)
	GetAll(ctx context.Context) ([]models.Chat, error)
	Delete(ctx context.Context, chatID int64) error
}

type Chat struct {
	repo ChatProvider
}

func NewChat(repo ChatProvider) *Chat {
	return &Chat{
		repo: repo,
	}
}

func (c *Chat) AddActive(ctx context.Context, chat *models.Chat) error {
	ch, _ := c.repo.GetByTitle(ctx, chat.Title)
	if ch != nil {
		return models.ErrAlreadyExist
	}

	if err := c.repo.Create(ctx, chat); err != nil {
		return fmt.Errorf("%w %w", models.ErrInternal, err)
	}

	return nil
}

func (c *Chat) Add(ctx context.Context, chat *models.Chat) error {
	ch, _ := c.repo.GetByTitle(ctx, chat.Title)
	if ch != nil {
		return models.ErrAlreadyExist
	}

	if err := c.repo.Create(ctx, chat); err != nil {
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

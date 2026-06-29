package service

import (
	"context"
	"fmt"
	"log/slog"

	"support_bot/internal/models"
)

type ChatProvider interface {
	Create(ctx context.Context, chat *models.TgChatDTO) error
	GetByTitle(ctx context.Context, title string) (*models.TgChatDTO, error)
	GetAll(ctx context.Context) ([]models.TgChatDTO, error)
	Delete(ctx context.Context, chatID int64) error
}

type Chat struct {
	repo   ChatProvider
	notify *Notify
	log    *slog.Logger
}

func NewChat(repo ChatProvider, notify *Notify, log *slog.Logger) *Chat {
	l := log.With(slog.Any("module", "tg_bot.service.chat"))

	return &Chat{
		repo:   repo,
		notify: notify,
		log:    l,
	}
}

func (c *Chat) AddActive(ctx context.Context, chat *models.TgChatDTO) error {
	ch, _ := c.repo.GetByTitle(ctx, chat.Title)
	if ch != nil {
		c.notify.SendAdminNotify(ctx, newAddNewChatErrorTemplate(*chat, models.ErrAlreadyExist))
		return models.ErrAlreadyExist
	}

	err := c.repo.Create(ctx, chat)
	if err != nil {
		c.notify.SendAdminNotify(ctx, newAddNewChatErrorTemplate(*chat, err))
		return fmt.Errorf("%w %w", models.ErrInternal, err)
	}

	c.notify.SendAdminNotify(ctx, newAddNewChatSuccessTemplate(*chat))

	return nil
}

func (c *Chat) Add(ctx context.Context, chat *models.TgChatDTO) error {
	ch, _ := c.repo.GetByTitle(ctx, chat.Title)
	if ch != nil {
		c.notify.SendAdminNotify(ctx, newAddNewChatErrorTemplate(*chat, models.ErrAlreadyExist))
		return models.ErrAlreadyExist
	}

	err := c.repo.Create(ctx, chat)
	if err != nil {
		c.notify.SendAdminNotify(ctx, newAddNewChatErrorTemplate(*chat, err))
		return fmt.Errorf("%w %w", models.ErrInternal, err)
	}

	c.notify.SendAdminNotify(ctx, newAddNewChatSuccessTemplate(*chat))
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

func (c *Chat) GetAll(ctx context.Context) ([]models.TgChatDTO, error) {
	chats, err := c.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

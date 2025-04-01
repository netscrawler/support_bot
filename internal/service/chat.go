package service

import (
	"context"
	"support_bot/internal/models"
	"support_bot/internal/repository"

	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

type Chat struct {
	repo *repository.Chat
	log  *zap.Logger
}

func newChat(repo *repository.Chat, log *zap.Logger) *Chat {
	return &Chat{
		repo: repo,
		log:  log,
	}
}

func (c *Chat) Add(ctx context.Context, chat *telebot.Chat) error {
	chatToSave := models.NewChat(chat)
	ch, _ := c.repo.GetByTitle(ctx, chat.Title)
	if ch != nil {
		return models.ErrAlreadyExist
	}
	if err := c.repo.Create(ctx, chatToSave); err != nil {
		return models.ErrInternal
	}
	return nil
}

func (c *Chat) Remove(ctx context.Context, title string) error {
	ch, err := c.repo.GetByTitle(ctx, title)
	if err != nil {
		return err
	}
	chID := ch.ChatID
	return c.repo.Delete(context.TODO(), chID)
}

func (c *Chat) GetAll(ctx context.Context) ([]models.Chat, error) {
	chats, err := c.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return chats, nil
}

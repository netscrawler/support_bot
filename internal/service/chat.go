package service

import (
	"context"
	"support_bot/internal/models"
	"support_bot/internal/repository"

	"go.uber.org/zap"
)

type Chat struct {
	repo *repository.Chat
	log  *zap.Logger
}

func NewChat(repo *repository.Chat, log *zap.Logger) *Chat {
	return &Chat{
		repo: repo,
		log:  log,
	}
}

func (c *Chat) GetChatByTitle(username string) (*models.Chat, error) {
	return nil, models.ErrInternal
}

func (c *Chat) Add(chat *models.Chat) error {
	if err := c.repo.Create(context.TODO(), chat); err != nil {
		return models.ErrInternal
	}
	return nil
}

func (c *Chat) Remove(title string) error {
	ch, err := c.repo.GetByTitle(context.TODO(), title)
	if err != nil {
		return err
	}
	chID := ch.ChatID
	return c.repo.Delete(context.TODO(), chID)
}

func (c *Chat) GetAll() ([]models.Chat, error) {
	chats, err := c.repo.GetAll(context.TODO())
	if err != nil {
		return nil, err
	}
	return chats, nil
}

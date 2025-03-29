package service

import (
	"context"
	"errors"
	"support_bot/internal/models"

	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

type ChatProvider interface {
	GetByTitle(ctx context.Context, title string) (*models.Chat, error)

	GetAll(ctx context.Context) ([]models.Chat, error)
}

type Notify struct {
	chat ChatProvider
	log  *zap.Logger
}

func NewNotify(c ChatProvider, log *zap.Logger) *Notify {
	return &Notify{
		chat: c,
		log:  log,
	}
}

func (n *Notify) Broadcast(ctx context.Context, bot *telebot.Bot, notify string) (int, error) {
	const op = "service.Notify.Broadcast"
	// count := 0
	chats, err := n.chat.GetAll(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return 0, models.ErrNotFound
		}
		return 0, models.ErrInternal

	}

	for _, chat := range chats {
		_, err := bot.Send(&telebot.Chat{ID: chat.ChatID}, notify)
		if err != nil {
			n.log.Error(op, zap.Error(err))
		}
	}

	return len(chats), nil
}

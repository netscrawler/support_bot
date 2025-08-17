package service

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/models"

	"gopkg.in/telebot.v4"
)

type ChatGetter interface {
	GetAll(ctx context.Context) ([]models.Chat, error)
	GetAllActive(ctx context.Context) ([]models.Chat, error)
}

type ChatNotify struct {
	chat      ChatGetter
	tgAdaptor MessageSender
	log       slog.Logger
}

func NewChatNotify(c ChatGetter, tgAdaptor MessageSender) *ChatNotify {
	return &ChatNotify{
		chat:      c,
		tgAdaptor: tgAdaptor,
		log:       *slog.Default(),
	}
}

// Broadcast При возникновении ошибки возвращает нули и ошибку.
func (n *ChatNotify) Broadcast(
	ctx context.Context,
	notify string,
) (string, error) {
	chats, err := n.chat.GetAllActive(ctx)
	if err != nil {
		n.log.ErrorContext(ctx, "error broadcast messages", slog.Any("error", err))
		if errors.Is(err, models.ErrNotFound) {
			return "", models.ErrNotFound
		}

		return "", models.ErrInternal
	}

	if len(chats) == 0 {
		return "", models.ErrNotFound
	}

	tgchats := []*telebot.Chat{}

	for _, chat := range chats {
		tgchat := telebot.Chat{ID: chat.ChatID, Title: chat.Title}
		tgchats = append(tgchats, &tgchat)
	}

	resp, err := n.tgAdaptor.Broadcast(tgchats, notify)

	return resp.String(), err
}

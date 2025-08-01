package service

import (
	"context"
	"errors"
	"support_bot/internal/models"

	"gopkg.in/telebot.v4"
)

type ChatGetter interface {
	GetAll(ctx context.Context) ([]models.Chat, error)
}

type ChatNotify struct {
	chat      ChatGetter
	tgAdaptor MessageSender
}

func NewChatNotify(c ChatGetter, tgAdaptor MessageSender) *ChatNotify {
	return &ChatNotify{
		chat:      c,
		tgAdaptor: tgAdaptor,
	}
}

// Broadcast При возникновении ошибки возвращает нули и ошибку.
func (n *ChatNotify) Broadcast(
	ctx context.Context,
	notify string,
) (string, error) {
	chats, err := n.chat.GetAll(ctx)
	if err != nil {
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

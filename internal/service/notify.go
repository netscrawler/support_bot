package service

import (
	"context"
	"errors"
	"support_bot/internal/models"

	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

type ChatProvider interface {
	GetAll(ctx context.Context) ([]models.Chat, error)
}

type Notify struct {
	chat ChatProvider
	log  *zap.Logger
}

func newNotify(c ChatProvider, log *zap.Logger) *Notify {
	return &Notify{
		chat: c,
		log:  log,
	}
}

// Отправляет уведомление во все чаты, возвращает количество чатов, количество успешных, количество ошибок, ошибку если возникла.
// При возникновении ошибки возвращает нули и ошибку
func (n *Notify) Broadcast(
	ctx context.Context,
	bot *telebot.Bot,
	notify string,
) (string, error) {
	const op = "service.Notify.Broadcast"
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

	resp := models.NewBroadcastResp()
	for _, chat := range chats {
		_, err := bot.Send(&telebot.Chat{ID: chat.ChatID}, notify)
		if err != nil {
			n.log.Error(op, zap.Error(err))
			resp.AddError(chat.Title)
			continue
		}
		resp.AddSuccess()
	}

	return resp.String(), nil
}

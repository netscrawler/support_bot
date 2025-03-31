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

type UserProvider interface {
	GetAll(ctx context.Context) ([]models.User, error)
	GetAllAdmins(ctx context.Context) ([]models.User, error)
}

type ChatNotify struct {
	chat ChatProvider
	log  *zap.Logger
}

func newChatNotify(c ChatProvider, log *zap.Logger) *ChatNotify {
	return &ChatNotify{
		chat: c,
		log:  log,
	}
}

// Отправляет уведомление во все чаты, возвращает количество чатов, количество успешных, количество ошибок, ошибку если возникла.
// При возникновении ошибки возвращает нули и ошибку
func (n *ChatNotify) Broadcast(
	ctx context.Context,
	bot *telebot.Bot,
	notify string,
) (string, error) {
	const op = "service.ChatNotify.Broadcast"
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

type UserNotify struct {
	user UserProvider
	log  *zap.Logger
}

func newUserNotify(up UserProvider, log *zap.Logger) *UserNotify {
	return &UserNotify{
		user: up,
		log:  log,
	}
}

func (n *UserNotify) Broadcast(ctx context.Context, bot *telebot.Bot, notify string) error {
	const op = "service.UserNotify.Broadcast"

	chats, err := n.user.GetAll(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrNotFound
		}
		return models.ErrInternal

	}

	for _, chat := range chats {
		_, err := bot.Send(&telebot.Chat{ID: chat.TelegramID}, notify)
		if err != nil {
			n.log.Error(op, zap.Error(err))
			continue
		}
	}

	return nil
}

func (n *UserNotify) SendNotify(
	ctx context.Context,
	bot *telebot.Bot,
	tgId int64,
	notify string,
) error {
	const op = "service.UserNotify.SendNotify"
	_, err := bot.Send(&telebot.Chat{ID: tgId}, notify)
	if err != nil {
		n.log.Error(op, zap.Error(err))
		return err
	}
	return nil
}

func (n *UserNotify) SendAdminNotify(ctx context.Context, bot *telebot.Bot, notify string) error {
	const op = "service.UserNotify.SendAdminNotify"

	chats, err := n.user.GetAllAdmins(ctx)
	n.log.Info(op, zap.Any("chats", chats))
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrNotFound
		}
		return models.ErrInternal

	}

	for _, user := range chats {
		_, err := bot.Send(&telebot.Chat{ID: user.TelegramID}, notify, telebot.ModeMarkdownV2)
		if err != nil {
			n.log.Error(op, zap.Error(err))
			continue
		}
	}

	return nil
}

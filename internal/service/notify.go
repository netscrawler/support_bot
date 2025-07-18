package service

import (
	"context"
	"errors"

	"support_bot/internal/models"

	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

type ChatGetter interface {
	GetAll(ctx context.Context) ([]models.Chat, error)
}

type UserGetter interface {
	GetAll(ctx context.Context) ([]models.User, error)
	GetAllAdmins(ctx context.Context) ([]models.User, error)
}

type MessageSender interface {
	Broadcast(chats []*telebot.Chat, msg string, opts ...interface{}) (*models.BroadcastResp, error)
	Send(chat *telebot.Chat, msg string, opts ...interface{}) error
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

// При возникновении ошибки возвращает нули и ошибку.
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

type UserNotify struct {
	user      UserGetter
	log       *zap.Logger
	tgAdaptor MessageSender
}

func NewUserNotify(up UserGetter, tgAdaptor MessageSender) *UserNotify {
	return &UserNotify{
		user:      up,
		tgAdaptor: tgAdaptor,
	}
}

func (n *UserNotify) Broadcast(ctx context.Context, notify string) error {
	users, err := n.user.GetAll(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrNotFound
		}

		return models.ErrInternal
	}

	tgchats := []*telebot.Chat{}

	for _, user := range users {
		tgchat := telebot.Chat{ID: user.TelegramID}
		tgchats = append(tgchats, &tgchat)
	}

	_, err = n.tgAdaptor.Broadcast(tgchats, notify)

	return err
}

func (n *UserNotify) SendNotify(
	ctx context.Context,
	tgID int64,
	notify string,
) error {
	const op = "service.UserNotify.SendNotify"

	err := n.tgAdaptor.Send(&telebot.Chat{ID: tgID}, notify)
	if err != nil {
		n.log.Error(op, zap.Error(err))

		return err
	}

	return nil
}

func (n *UserNotify) SendAdminNotify(ctx context.Context, bot *telebot.Bot, notify string) error {
	const op = "service.UserNotify.SendAdminNotify"

	users, err := n.user.GetAllAdmins(ctx)
	n.log.Info(op, zap.Any("chats", users))

	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrNotFound
		}

		return models.ErrInternal
	}

	tgchats := []*telebot.Chat{}

	for _, user := range users {
		tgchat := telebot.Chat{ID: user.TelegramID}
		tgchats = append(tgchats, &tgchat)
	}

	_, err = n.tgAdaptor.Broadcast(tgchats, notify, telebot.ModeMarkdownV2)

	return err
}

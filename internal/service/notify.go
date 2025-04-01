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

type MessageSender interface {
	Broadcast(chats []*telebot.Chat, msg string, opts ...interface{}) (*models.BroadcastResp, error)
	Send(chat *telebot.Chat, msg string, opts ...interface{}) error
}

type ChatNotify struct {
	chat      ChatProvider
	log       *zap.Logger
	tgAdaptor MessageSender
}

func newChatNotify(c ChatProvider, log *zap.Logger, tgAdaptor MessageSender) *ChatNotify {
	return &ChatNotify{
		chat:      c,
		log:       log,
		tgAdaptor: tgAdaptor,
	}
}

// Отправляет уведомление во все чаты, возвращает количество чатов, количество успешных, количество ошибок, ошибку если возникла.
// При возникновении ошибки возвращает нули и ошибку
func (n *ChatNotify) Broadcast(
	ctx context.Context,
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

	tgchats := []*telebot.Chat{}
	for _, chat := range chats {
		tgchat := telebot.Chat{ID: chat.ChatID, Title: chat.Title}
		tgchats = append(tgchats, &tgchat)
	}

	resp, err := n.tgAdaptor.Broadcast(tgchats, notify)
	return resp.String(), err
}

type UserNotify struct {
	user      UserProvider
	log       *zap.Logger
	tgAdaptor MessageSender
}

func newUserNotify(up UserProvider, log *zap.Logger, tgAdaptor MessageSender) *UserNotify {
	return &UserNotify{
		user:      up,
		log:       log,
		tgAdaptor: tgAdaptor,
	}
}

func (n *UserNotify) Broadcast(ctx context.Context, notify string) error {
	const op = "service.UserNotify.Broadcast"

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
	tgId int64,
	notify string,
) error {
	const op = "service.UserNotify.SendNotify"
	err := n.tgAdaptor.Send(&telebot.Chat{ID: tgId}, notify)
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

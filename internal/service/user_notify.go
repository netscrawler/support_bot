package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/models"

	"gopkg.in/telebot.v4"
)

type UserGetter interface {
	GetAll(ctx context.Context) ([]models.User, error)
	GetAllAdmins(ctx context.Context) ([]models.User, error)
}

type MessageSender interface {
	Broadcast(chats []*telebot.Chat, msg string, opts ...any) (*models.BroadcastResp, error)
	Send(chat *telebot.Chat, msg string, opts ...any) error
	SendDocument(
		chat *telebot.Chat,
		buf *bytes.Buffer,
		filename string,
		opts ...any,
	) error
	SendMedia(chat *telebot.Chat, imgs []*bytes.Buffer, opts ...any) error
}

type UserNotify struct {
	user UserGetter
	// log       *zap.Logger
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

	l := slog.Default()

	err := n.tgAdaptor.Send(&telebot.Chat{ID: tgID}, notify)
	if err != nil {
		l.ErrorContext(ctx, "Send notify error", slog.Any("error", err))

		return err
	}

	return nil
}

func (n *UserNotify) SendAdminNotify(ctx context.Context, bot *telebot.Bot, notify string) error {
	const op = "service.UserNotify.SendAdminNotify"

	users, err := n.user.GetAllAdmins(ctx)
	// n.log.Info(op, zap.Any("chats", users))
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

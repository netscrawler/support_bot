package service

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/models"
)

type UserGetter interface {
	GetAll(ctx context.Context) ([]models.User, error)
	GetAllAdmins(ctx context.Context) ([]models.User, error)
}

type MessageSender interface {
	Broadcast(chats []models.Chat, msg string, opts ...any) (*models.BroadcastResp, error)
	Send(chat models.Chat, msg string, opts ...any) error
}

type UserNotify struct {
	user      UserGetter
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

	tgchats := []models.Chat{}

	for _, user := range users {
		tgchat := models.Chat{ChatID: user.TelegramID}
		tgchats = append(tgchats, tgchat)
	}

	_, err = n.tgAdaptor.Broadcast(tgchats, notify)

	return err
}

func (n *UserNotify) SendNotify(
	ctx context.Context,
	tgID int64,
	notify string,
) error {
	l := slog.Default()

	err := n.tgAdaptor.Send(models.Chat{ChatID: tgID}, notify)
	if err != nil {
		l.ErrorContext(ctx, "Send notify error", slog.Any("error", err))

		return err
	}

	return nil
}

func (n *UserNotify) SendAdminNotify(ctx context.Context, notify string) error {
	users, err := n.user.GetAllAdmins(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrNotFound
		}

		return models.ErrInternal
	}

	tgchats := []models.Chat{}

	for _, user := range users {
		tgchat := models.Chat{ChatID: user.TelegramID}
		tgchats = append(tgchats, tgchat)
	}

	_, err = n.tgAdaptor.Broadcast(tgchats, notify, models.ModeMarkdownV2)

	return err
}

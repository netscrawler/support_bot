package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"support_bot/internal/models"
)

type UserGetter interface {
	GetAll(ctx context.Context) ([]models.User, error)
	GetAllAdmins(ctx context.Context) ([]models.User, error)
}

type ActiveChatGetter interface {
	GetAllActive(ctx context.Context) ([]models.Chat, error)
}

type TelegramNotify struct {
	user   UserGetter
	chat   ActiveChatGetter
	sender Sender
}

func NewTelegramNotify(up UserGetter, chatGetter ActiveChatGetter, sender Sender) *TelegramNotify {
	return &TelegramNotify{
		user:   up,
		sender: sender,
		chat:   chatGetter,
	}
}

func (n *TelegramNotify) BroadcastToUsers(ctx context.Context, notify string) error {
	users, err := n.user.GetAll(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrNotFound
		}

		return models.ErrInternal
	}

	var jerr error

	for _, user := range users {
		lerr := n.SendNotify(ctx, user.TelegramID, notify)
		jerr = errors.Join(jerr, lerr)
	}

	return jerr
}

func (n *TelegramNotify) SendNotify(
	ctx context.Context,
	tgID int64,
	notify string,
) error {
	l := slog.Default()

	err := n.sender.Send(models.NewTargetTelegramChat(tgID, nil), models.NewTextData(notify))
	if err != nil {
		l.ErrorContext(ctx, "Send notify error", slog.Any("error", err))
	}

	return err
}

func (n *TelegramNotify) SendAdminNotify(ctx context.Context, notify string) error {
	users, err := n.user.GetAllAdmins(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrNotFound
		}

		return models.ErrInternal
	}

	var jerr error

	for _, user := range users {
		lerr := n.SendNotify(ctx, user.TelegramID, notify)
		jerr = errors.Join(jerr, lerr)
	}

	return jerr
}

func (n *TelegramNotify) BroadcastToChats(
	ctx context.Context,
	notify string,
) (string, error) {
	l := slog.Default()

	chats, err := n.chat.GetAllActive(ctx)
	if err != nil {
		l.ErrorContext(ctx, "error broadcast messages", slog.Any("error", err))

		if errors.Is(err, models.ErrNotFound) {
			return "", models.ErrNotFound
		}

		return "", fmt.Errorf("%w %w", models.ErrInternal, err)
	}

	if len(chats) == 0 {
		return "", models.ErrNotFound
	}

	resp := models.NewBroadcastResp()

	sendData := models.NewTextData(notify)

	for _, chat := range chats {
		err := n.sender.Send(
			models.NewTargetTelegramChat(chat.ChatID, nil),
			sendData,
		)
		if err != nil {
			resp.AddError(chat.Title, err)

			continue
		}

		resp.AddSuccess()
	}

	return resp.String(), nil
}

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"support_bot/internal/delivery/telegram"
	models "support_bot/internal/models/notify"
	report "support_bot/internal/models/report"
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
	sender *telegram.ChatAdaptor
	log    *slog.Logger
}

func NewTelegramNotify(
	up UserGetter,
	chatGetter ActiveChatGetter,
	sender *telegram.ChatAdaptor,
	log *slog.Logger,
) *TelegramNotify {
	l := log.With(slog.Any("module", "tg_bot.service.notify"))

	return &TelegramNotify{
		user:   up,
		sender: sender,
		chat:   chatGetter,
		log:    l,
	}
}

func (n *TelegramNotify) BroadcastToUsers(ctx context.Context, notify string) error {
	n.log.DebugContext(ctx, "start broadcasting to users", slog.Any("notify", notify))

	users, err := n.user.GetAll(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			n.log.InfoContext(ctx, "not found users")

			return models.ErrNotFound
		}

		n.log.ErrorContext(ctx, "error while loading users", slog.Any("error", err))

		return models.ErrInternal
	}

	var jerr error

	for _, user := range users {
		lerr := n.SendNotify(ctx, user.TelegramID, notify)
		jerr = errors.Join(jerr, lerr)
	}

	n.log.InfoContext(ctx, "finish broadcasting", slog.Any("error", jerr))

	return jerr
}

func (n *TelegramNotify) SendNotify(
	ctx context.Context,
	tgID int64,
	notify string,
) error {
	err := n.sender.Send(
		ctx,
		report.NewTargetTelegramChat(tgID, nil),
		[]report.ReportData{report.NewTextData(notify)}...)
	if err != nil {
		n.log.ErrorContext(
			ctx,
			"Send notify error",
			slog.Any("error", err),
			slog.Any("user_id", tgID),
		)
	}

	return err
}

func (n *TelegramNotify) SendAdminNotify(ctx context.Context, notify string) error {
	n.log.DebugContext(ctx, "start broadcasting to admins", slog.Any("notify", notify))

	users, err := n.user.GetAllAdmins(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			n.log.InfoContext(ctx, "not found admins")

			return models.ErrNotFound
		}

		n.log.ErrorContext(ctx, "error while loading admins", slog.Any("error", err))

		return models.ErrInternal
	}

	var jerr error

	for _, user := range users {
		lerr := n.SendNotify(ctx, user.TelegramID, notify)
		jerr = errors.Join(jerr, lerr)
	}

	n.log.InfoContext(ctx, "finish broadcasting", slog.Any("error", jerr))

	return jerr
}

func (n *TelegramNotify) BroadcastToChats(
	ctx context.Context,
	notify string,
) (string, error) {
	n.log.DebugContext(ctx, "start broadcasting to chats", slog.Any("notify", notify))

	chats, err := n.chat.GetAllActive(ctx)
	if err != nil {
		n.log.ErrorContext(ctx, "error broadcast messages", slog.Any("error", err))

		if errors.Is(err, models.ErrNotFound) {
			return "", models.ErrNotFound
		}

		return "", fmt.Errorf("%w %w", models.ErrInternal, err)
	}

	if len(chats) == 0 {
		return "", models.ErrNotFound
	}

	resp := models.NewBroadcastResp()

	sendData := report.NewTextData(notify)

	for _, chat := range chats {
		err := n.sender.Send(
			ctx,
			report.NewTargetTelegramChat(chat.ChatID, nil),
			[]report.ReportData{sendData}...,
		)
		if err != nil {
			resp.AddError(chat.Title, err)

			continue
		}

		resp.AddSuccess()
	}

	n.log.InfoContext(
		ctx,
		"finish broadcasting to chats",
		slog.Any("chats_count", resp.ChatsCount),
		slog.Any("success count", resp.SuccessCount),
		slog.Any("error count", resp.ErrorCount),
		slog.Any("error", resp.ErrorChatsTitles),
	)

	return resp.String(), nil
}

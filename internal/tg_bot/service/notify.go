package service

import (
	"context"
	"log/slog"

	"support_bot/internal/delivery/telegram"
	"support_bot/internal/models"
	"support_bot/internal/tg_bot/repository"
)

type Notify struct {
	tg   *telegram.ChatAdaptor
	user *repository.UserRepository
	log  *slog.Logger
}

func NewNotify(tg *telegram.ChatAdaptor, user *repository.UserRepository, log *slog.Logger) *Notify {
	return &Notify{tg: tg, user: user, log: log}
}

func (n *Notify) SendAdminNotify(ctx context.Context, msg string) {
	admins, err := n.user.GetAllAdmins(ctx)
	if err != nil {
		n.log.ErrorContext(
			ctx,
			"unable to get all admins",
			slog.Any("error", err),
		)

		return
	}

	if len(admins) == 0 {
		n.log.ErrorContext(ctx, "admins not found")
		return
	}

	for _, admin := range admins {
		_, err := n.tg.SendText(ctx, models.TgChat{ChatID: admin.TelegramID}, msg)
		if err != nil {
			n.log.ErrorContext(
				ctx,
				"failed to send admin notify",
				slog.Any("admin", admin),
				slog.Any("error", err),
			)
		}
	}
}

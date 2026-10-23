package runlog

import (
	"context"
	"support_bot/internal/models"
)

type Alerter interface {
	Alert(ctx context.Context, text string) error
}

type TextSender interface {
	SendText(ctx context.Context, rcpt models.TgChat, s string) (*models.SentMessage, error)
}

type AdminNotifier interface {
	SendAdminNotify(ctx context.Context, msg string)
}

type tgAlerter struct {
	cfg    AlertsConfig
	tg     TextSender
	admins AdminNotifier
}

// NewTgAlerter шлёт в alerts.chat_id, а если он не задан — всем админам в личку.
func NewTgAlerter(cfg AlertsConfig, tg TextSender, admins AdminNotifier) Alerter {
	return &tgAlerter{cfg: cfg, tg: tg, admins: admins}
}

func (a *tgAlerter) Alert(ctx context.Context, text string) error {
	if a.cfg.ChatID != 0 {
		_, err := a.tg.SendText(
			ctx,
			models.TgChat{ChatID: a.cfg.ChatID, ThreadID: a.cfg.ThreadID},
			text,
		)
		return err
	}
	a.admins.SendAdminNotify(ctx, text)
	return nil
}

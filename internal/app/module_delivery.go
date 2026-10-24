package app

import (
	"context"
	"log/slog"

	"support_bot/internal/config"
	"support_bot/internal/delivery"
	maxadp "support_bot/internal/delivery/max"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/delivery/telegram"

	maxcli "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/mymmrac/telego"
	"go.uber.org/fx"
)

var deliveryModule = fx.Module(
	"delivery",
	fx.Provide(
		newTelegramAdaptor,
		newMaxAdaptor,
		newSMTP,
		newSMB,
		newDispatcher,
	),
)

func newTelegramAdaptor(
	tgBot *telego.Bot,
	cfg *config.Config,
	log *slog.Logger,
) *telegram.ChatAdaptor {
	return telegram.NewChatAdaptor(tgBot, cfg.Delivery.Retry.Delays, log)
}

func newMaxAdaptor(
	maxBot *maxcli.Api,
	cfg *config.Config,
	log *slog.Logger,
) *maxadp.Adaptor {
	return maxadp.New(
		maxBot,
		cfg.Delivery.Retry.Delays,
		cfg.MaxBot.Enabled,
		log,
	)
}

func newSMTP(cfg *config.Config, log *slog.Logger) *smtp.Sender {
	return smtp.New(cfg.SMTP, cfg.Delivery.Retry.Delays, log)
}

func newSMB(
	ctx context.Context,
	cfg *config.Config,
	log *slog.Logger,
	lc fx.Lifecycle,
) (*smb.SMB, error) {
	smbS, err := smb.New(ctx, cfg.SMB, cfg.Delivery.Retry.Delays, log)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return smbS.Close()
		},
	})

	return smbS, nil
}

func newDispatcher(
	tg *telegram.ChatAdaptor,
	smbS *smb.SMB,
	smtpS *smtp.Sender,
	maxAdp *maxadp.Adaptor,
) *delivery.Dispatcher {
	return delivery.NewDispatcher(tg, smbS, smtpS, maxAdp)
}

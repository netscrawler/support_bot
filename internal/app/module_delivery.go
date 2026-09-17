package app

import (
	"context"
	"log/slog"
	"support_bot/internal/config"
	maxadp "support_bot/internal/delivery/max"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/delivery/telegram"
	"support_bot/internal/models"
	"support_bot/internal/pkg/retry"

	maxcli "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/mymmrac/telego"

	"go.uber.org/fx"
)

var deliveryModule = fx.Module(
	"delivery",
	fx.Provide(newTelegramAdaptor, newMaxAdaptor, newSMTP, newSMB, newSenderProvider),
)

func newTelegramAdaptor(tgBot *telego.Bot, retr *retry.Retry, log *slog.Logger) *telegram.ChatAdaptor {
	return telegram.NewChatAdaptor(tgBot, retr, log)
}

func newMaxAdaptor(maxBot *maxcli.Api, retr *retry.Retry, cfg *config.Config, log *slog.Logger) *maxadp.Adaptor {
	return maxadp.New(maxBot, retr, cfg.MaxBot.Enabled, log)
}

func newSMTP(cfg *config.Config, log *slog.Logger) *smtp.Sender {
	return smtp.New(cfg.SMTP, log)
}

func newSMB(ctx context.Context, cfg *config.Config, log *slog.Logger, lc fx.Lifecycle) (*smb.SMB, error) {
	smbS, err := smb.New(ctx, cfg.SMB, log)
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

func newSenderProvider(
	tg *telegram.ChatAdaptor,
	smbS *smb.SMB,
	smtpS *smtp.Sender,
	maxAdp *maxadp.Adaptor,
) *models.SenderProvider {
	return models.NewSenderProvider(tg, smbS, smtpS, maxAdp)
}

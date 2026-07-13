package tgbot

import (
	"log/slog"
	"net/http"
	"support_bot/internal/pkg"

	"gopkg.in/telebot.v4"
)

func NewTelegramBot(
	cfg Config,
	log *slog.Logger,
) (*telebot.Bot, error) {
	l := log.With("telegram_bot")

	client := &http.Client{}
	if cfg.Proxy != "" {
		l.Info(
			"proxy addr not empty, creating bot with system proxy",
			slog.Any("proxy addr", cfg.Proxy),
		)
		clientB, err := pkg.BuildHTTPClient(cfg.Proxy)
		if err != nil {
			return nil, err
		}
		client = clientB
	}

	pref := telebot.Settings{
		Token:  cfg.TelegramToken,
		Poller: &telebot.LongPoller{Timeout: cfg.BotPoll},
		Client: client,
	}

	if cfg.ApiProxy != "" {
		l.Info(
			"api proxy not empty, creating bot with custom api server",
			slog.Any("api server", cfg.ApiProxy),
		)
		pref.URL = cfg.ApiProxy
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return b, nil
}

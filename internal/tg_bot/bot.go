package tgbot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"support_bot/internal/pkg"
)

func NewTelegramBot(
	ctx context.Context,
	cfg Config,
	log *slog.Logger,
) (*telego.Bot, *th.BotHandler, error) {
	opts := []telego.BotOption{}

	if cfg.Proxy != "" {
		log.Info(
			"proxy addr not empty, creating bot with system proxy",
			slog.Any("proxy addr", cfg.Proxy),
		)

		client, err := pkg.BuildHTTPClient(cfg.Proxy)
		if err != nil {
			return nil, nil, err
		}

		opts = append(opts, telego.WithHTTPClient(client))
	}

	if cfg.ApiProxy != "" {
		log.Info(
			"api proxy not empty, creating bot with custom api server",
			slog.Any("api server", cfg.ApiProxy),
		)
		opts = append(opts, telego.WithAPIServer(cfg.ApiProxy))
	}

	tgBot, err := telego.NewBot(cfg.TelegramToken, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("create telegram bot: %w", err)
	}

	botUser, err := tgBot.GetMe(ctx)
	if err != nil {
		return tgBot, nil, fmt.Errorf("get bot user: %w", err)
	}

	log.InfoContext(ctx, "telegram bot authorized as", slog.Any("user", botUser))

	updates, err := tgBot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		return tgBot, nil, fmt.Errorf("long polling: %w", err)
	}

	bh, err := th.NewBotHandler(tgBot, updates)
	if err != nil {
		return tgBot, nil, fmt.Errorf("bot handler: %w", err)
	}

	return tgBot, bh, nil
}

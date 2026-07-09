package maxbot

import (
	"context"
	"fmt"
	"log/slog"

	maxcli "github.com/max-messenger/max-bot-api-client-go/v2"
)

func New(ctx context.Context, cfg Config, log *slog.Logger) (*maxcli.Api, error) {
	var opts []maxcli.Opt
	l := log.WithGroup("max")

	if cfg.BotPoll != 0 {
		l.InfoContext(ctx, "setting bot poll to ", slog.Any("poll", cfg.BotPoll))
		opts = append(opts, maxcli.WithPollingTimeout(cfg.BotPoll))
	}

	if cfg.ApiProxy != "" {
		l.InfoContext(
			ctx,
			"api proxy from config not empty, creating bot with base url",
			slog.Any("url", cfg.ApiProxy),
		)
		opts = append(opts, maxcli.WithBaseURL(cfg.ApiProxy))
	}

	api, err := maxcli.NewApi(cfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("create max api: %w", err)
	}

	info, err := api.Bots.GetMyInfo(ctx)
	if err != nil {
		l.ErrorContext(ctx, "creating bot: getting bot info", slog.Any("error", err))
		return nil, fmt.Errorf("creating bot: get bot info: %w", err)
	}

	l.InfoContext(
		ctx,
		"logged as",
		slog.Group(
			"bot",
			slog.Any("user_id", info.UserID),
			slog.Any("username", info.Username),
			slog.Any("first_name", info.FirstName),
		),
	)

	return api, nil
}

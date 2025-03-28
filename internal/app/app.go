package app

import (
	"context"
	"fmt"
	"support_bot/internal/app/bot"
	"support_bot/internal/config"
	"support_bot/internal/database/postgres"

	"go.uber.org/zap"
)

type App struct {
	bot *bot.Bot
	log *zap.Logger
	cfg *config.Config
}

func New(ctx context.Context, cfg *config.Config, log *zap.Logger) (*App, error) {
	s := postgres.New(log)

	err := s.Init(ctx, cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	b, err := bot.New(
		cfg.Bot.TelegramToken,
		cfg.Timeout.BotPoll,
		log,
	)
	if err != nil {
		return nil, err
	}

	return &App{
		bot: b,
		log: log,
		cfg: cfg,
	}, nil
}

func (a *App) Start() error {
	err := a.bot.Start()
	if err != nil {
		return err
	}

	return nil
}

func (a *App) GracefulShutdown() {
	const op = "app.GracefulShutdown"
	a.log.Info(fmt.Sprintf("%s : shutting down application", op))
	a.bot.Stop()

	a.log.Info(fmt.Sprintf("%s : application stopped", op))
}

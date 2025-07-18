package app

import (
	"context"

	"support_bot/internal/app/bot"
	"support_bot/internal/config"
	postgres "support_bot/internal/infra/out/pg"

	"github.com/jackc/pgx/v5"
)

type App struct {
	bot     *bot.Bot
	storage *pgx.Conn
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	conn, err := postgres.New(context.TODO(), cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	b, err := bot.New(
		cfg.Bot.TelegramToken,
		cfg.Timeout.BotPoll,
		cfg.Bot.CleanUpTime,
		conn)
	if err != nil {
		return nil, err
	}

	return &App{
		bot:     b,
		storage: conn,
	}, nil
}

func (a *App) Start() error {
	err := a.bot.Start()
	if err != nil {
		return err
	}

	return nil
}

func (a *App) GracefulShutdown(ctx context.Context) {
	const op = "app.GracefulShutdown"

	// a.log.Info(op + " : shutting down application")
	// a.bot.Stop()
	// a.storage.Close(ctx)
	// a.log.Info(op + " : application stopped")
}

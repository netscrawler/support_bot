package app

import (
	"context"
	"log/slog"
	"support_bot/internal/app/bot"
	"support_bot/internal/app/report"
	"support_bot/internal/config"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"
	"support_bot/internal/sheduler"

	"gopkg.in/telebot.v4"
)

type App struct {
	bot     *telebot.Bot
	storage *postgres.DB
	cfg     *config.Config
	report  *report.App
	tgBot   *bot.Bot
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	connCtx, cancel := context.WithTimeout(ctx, cfg.Database.DatabaseConnect)
	defer cancel()

	log := slog.Default()

	connCtx = logger.AppendCtx(connCtx, slog.Any("function", "connecting to database"))

	rdb, err := postgres.New(ctx, cfg.Database, log)
	if err != nil {
		log.ErrorContext(connCtx, "unable to create connection", slog.Any("error", err))

		return nil, err
	}

	tgBot, err := bot.NewTgBot(cfg.Bot.TelegramToken, cfg.Bot.BotPoll)
	if err != nil {
		return nil, err
	}

	shdAPI := make(chan sheduler.SheduleAPIEvent, 5)

	report, err := report.New(ctx, cfg, tgBot, rdb, shdAPI, log)
	if err != nil {
		return nil, err
	}

	tgBotUser, err := bot.New(cfg.Bot.CleanUpTime, tgBot, rdb, shdAPI, log)
	if err != nil {
		return nil, err
	}

	return &App{
		bot:     tgBot,
		storage: rdb,
		cfg:     cfg,
		report:  report,
		tgBot:   tgBotUser,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	go a.bot.Start()
	go a.tgBot.Start()

	return a.report.Start(ctx)
}

func (a *App) GracefulShutdown(ctx context.Context) {
	log := slog.Default()
	log.InfoContext(ctx, "start")

	a.tgBot.Stop()
	a.bot.Stop()
	a.report.Stop(ctx)
	log.InfoContext(ctx, "bot stopped")

	err := a.storage.Stop(ctx)
	if err != nil {
		log.ErrorContext(ctx, "unable to close db connection", slog.Any("error", err))

		return
	}

	log.InfoContext(ctx, "successfully stop")
}

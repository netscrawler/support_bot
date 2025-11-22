package app

import (
	"context"
	"log/slog"

	"support_bot/internal/app/bot"
	"support_bot/internal/config"
	"support_bot/internal/infra/out/metabase"
	postgres "support_bot/internal/infra/out/pg"
	pgrepo "support_bot/internal/infra/out/pg/repo"
	"support_bot/internal/infra/out/smb"
	telegram "support_bot/internal/infra/out/tg"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/service"
)

type App struct {
	bot     *bot.Bot
	storage *postgres.ReconnectableDB
	cfg     *config.Config
	stats   *service.Report
	smb     *smb.SMB
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	connCtx, cancel := context.WithTimeout(ctx, cfg.Timeout.DatabaseConnect)
	defer cancel()

	log := slog.Default()

	connCtx = logger.AppendCtx(connCtx, slog.Any("function", "connecting to database"))

	rdb, err := postgres.New(ctx, cfg.Database.URL)
	if err != nil {
		log.InfoContext(connCtx, "unable to create connection", slog.Any("error", err))

		return nil, err
	}

	var smbConn *smb.SMB

	if cfg.SMB.Active {
		smbConn, err = smb.New(
			ctx,
			cfg.SMB.Address,
			cfg.SMB.User,
			cfg.SMB.PWD,
			cfg.SMB.Domain,
			cfg.SMB.Share,
		)
		if err != nil {
			log.ErrorContext(ctx, "unable to connect to smb", slog.Any("error", err))

			return nil, err
		}
	}

	tgBot, err := bot.NewTgBot(cfg.Bot.TelegramToken, cfg.Timeout.BotPoll)
	if err != nil {
		return nil, err
	}

	chatRepo := pgrepo.NewChat(rdb)
	userRepo := pgrepo.NewUser(rdb)
	notifyRepo := pgrepo.NewQuery(rdb)

	chatService := service.NewChat(chatRepo)
	userService := service.NewUser(userRepo)

	tgSender := telegram.NewChatAdaptor(tgBot)
	senderStrategy := service.NewSender(tgSender, smbConn)

	userNotifier := service.NewTelegramNotify(userRepo, chatRepo, senderStrategy)

	// Создаем Metabase клиент
	metabaseClient := metabase.New(cfg.MetabaseDomain)
	statsService := service.New(notifyRepo, senderStrategy, metabaseClient)

	b, err := bot.New(
		cfg.Bot.CleanUpTime,
		tgBot,
		userService,
		chatService,
		userNotifier,
		statsService,
	)
	if err != nil {
		return nil, err
	}

	return &App{
		bot:     b,
		storage: rdb,
		cfg:     cfg,
		stats:   statsService,
		smb:     smbConn,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.bot.Start()

	log := slog.Default()

	if _, err := a.stats.Start(context.Background()); err != nil {
		log.Error("unable start cron", slog.Any("error", err))
	}

	log.Info("application successfully started")

	return nil
}

func (a *App) GracefulShutdown(ctx context.Context) {
	log := slog.Default()
	log.InfoContext(ctx, "start")
	a.stats.Stop()

	log.InfoContext(ctx, "cron stopped")

	a.bot.Stop()
	log.InfoContext(ctx, "bot stopped")

	if a.smb != nil {
		err := a.smb.Close()
		if err != nil {
			log.ErrorContext(ctx, "unable to close smb connection", slog.Any("error", err))
		}
	}

	err := a.storage.Stop(ctx)
	if err != nil {
		log.ErrorContext(ctx, "unable to close db connection", slog.Any("error", err))

		return
	}

	log.InfoContext(ctx, "successfully stop")
}

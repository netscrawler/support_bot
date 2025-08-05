package app

import (
	"context"
	"log/slog"
	"math/rand"
	"net/http"
	"support_bot/internal/app/bot"
	"support_bot/internal/config"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/service"
	"time"

	postgres "support_bot/internal/infra/out/pg"
	pgrepo "support_bot/internal/infra/out/pg/repo"
	telegram "support_bot/internal/infra/out/tg"

	"github.com/jackc/pgx/v5"
)

type App struct {
	bot     *bot.Bot
	storage *pgx.Conn
	cfg     *config.Config
	stats   *service.Stats
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	connCtx, cancel := context.WithTimeout(ctx, cfg.Timeout.DatabaseConnect)
	defer cancel()

	log := slog.Default()

	connCtx = logger.AppendCtx(connCtx, slog.Any("function", "connecting to database"))

	conn, err := postgres.New(connCtx, cfg.Database.URL)
	if err != nil {
		log.InfoContext(connCtx, "unable to create connection", slog.Any("error", err))

		return nil, err
	}

	app := &App{
		cfg:     cfg,
		storage: conn,
	}

	tgBot, err := bot.NewTgBot(cfg.Bot.TelegramToken, cfg.Timeout.BotPoll)
	if err != nil {
		return nil, err
	}

	recDB := postgres.NewReconnectableDB(func() *pgx.Conn {
		return app.storage
	})

	chatRepo := pgrepo.NewChat(recDB)
	userRepo := pgrepo.NewUser(recDB)
	notifyRepo := pgrepo.NewQuery(recDB)

	chatService := service.NewChat(chatRepo)
	userService := service.NewUser(userRepo)

	messageSender := telegram.NewChatAdaptor(tgBot)

	notifyier := service.NewChatNotify(chatRepo, messageSender)
	userNotifier := service.NewUserNotify(userRepo, messageSender)

	// Создаем Metabase клиент
	metabaseClient := service.NewMetabase(cfg.MetabaseDomain, &http.Client{})
	statsService := service.New(notifyRepo, messageSender, metabaseClient)

	b, err := bot.New(
		cfg.Bot.CleanUpTime,
		tgBot,
		userService,
		chatService,
		notifyier,
		userNotifier,
		statsService,
	)
	if err != nil {
		return nil, err
	}

	app.bot = b
	app.stats = statsService

	return app, nil
}

func (a *App) Start(ctx context.Context) error {
	a.bot.Start()

	log := slog.Default()

	if err := a.stats.Start(context.Background()); err != nil {
		log.Error("unable start cron", slog.Any("error", err))
	}

	a.startDBMonitor(ctx)

	log.Info("application successfully started")

	return nil
}

func (a *App) startDBMonitor(ctx context.Context) {
	go func() {
		log := slog.Default()
		log.InfoContext(ctx, "starting db monitor")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("DB monitor stopped", slog.String("reason", ctx.Err().Error()))
				return
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := a.storage.Ping(pingCtx)
				cancel()

				if err != nil {
					log.Warn("DB ping failed, starting reconnect loop", slog.Any("error", err))
					a.reconnectLoop(ctx)
				}
			}
		}
	}()
}

func (a *App) reconnectLoop(ctx context.Context) {
	log := slog.Default()

	const (
		initialDelay = time.Second      // стартовая задержка
		maxDelay     = time.Minute      // максимальная задержка
		timeoutEach  = 10 * time.Second // таймаут на одну попытку пинга или реконнекта
	)

	delay := initialDelay

	for {
		select {
		case <-ctx.Done():
			log.Info("Reconnect loop stopped", slog.String("reason", ctx.Err().Error()))
			return
		default:
			connCtx, cancel := context.WithTimeout(ctx, timeoutEach)
			err := a.storage.Ping(connCtx)
			cancel()

			if err == nil {
				log.Info("Database connection restored")
				return
			}

			log.Warn("Database unavailable", slog.Any("error", err))

			if a.storage.IsClosed() {
				connCtx, cancel := context.WithTimeout(ctx, timeoutEach)
				newConn, err := postgres.New(connCtx, a.cfg.Database.URL)
				cancel()
				if err == nil {
					a.storage = newConn
					log.Info("Successfully reconnected to database")
					return
				}
				log.Warn("Reconnect attempt failed", slog.Any("error", err))
			}

			sleep := time.Duration(rand.Int63n(int64(delay)))
			log.Info("Waiting before next retry", slog.Duration("delay", sleep))
			time.Sleep(sleep)

			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

func (a *App) GracefulShutdown(ctx context.Context) {
	log := slog.Default()
	log.InfoContext(ctx, "start")
	a.stats.Stop()

	log.InfoContext(ctx, "cron stopped")

	a.bot.Stop()
	log.InfoContext(ctx, "bot stopped")

	err := a.storage.Close(ctx)
	if err != nil {
		log.InfoContext(ctx, "unable to close db connection", slog.Any("error", err))

		return
	}

	log.InfoContext(ctx, "successfully stop")
}

package bot

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"support_bot/internal/delivery/telegram"
	"support_bot/internal/postgres"
	"support_bot/internal/sheduler"
	bot "support_bot/internal/tg_bot"
	"support_bot/internal/tg_bot/handlers"
	"support_bot/internal/tg_bot/middlewares"
	"support_bot/internal/tg_bot/repository"
	"support_bot/internal/tg_bot/service"

	"gopkg.in/telebot.v4"
)

type Bot struct {
	bot    *telebot.Bot
	router *bot.Router

	shed *sheduler.SheduleAPI

	// goroutine lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewTgBot(token string, poll time.Duration) (*telebot.Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poll},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func New(
	cleanupTime time.Duration,
	tgBot *telebot.Bot,
	db *postgres.DB,
	shdAPI chan sheduler.SheduleAPIEvent,
	log *slog.Logger,
) (*Bot, error) {
	state := handlers.NewState(cleanupTime)

	chatRepo := repository.NewChatRepository(db.GetConn(), log)
	userRepo := repository.NewUserRepository(db.GetConn(), log)

	chatService := service.NewChat(chatRepo)
	userService := service.NewUser(userRepo)

	tgSender := telegram.NewChatAdaptor(tgBot, log)
	shed := sheduler.NewSheduleAPI(shdAPI)
	reportService := service.NewReportService(shed, log)

	notifyier := service.NewTelegramNotify(userRepo, chatRepo, tgSender)

	adminHandler := handlers.NewAdminHandler(
		tgBot,
		userService,
		chatService,
		notifyier,
		reportService,
		state,
	)

	userHandler := handlers.NewUserHandler(
		tgBot,
		chatService,
		userService,
		state,
		notifyier,
	)

	textHandler := handlers.NewTextHandler(adminHandler, userHandler, state)

	mw := middlewares.NewMw(userService)

	router := bot.NewRouter(tgBot, adminHandler, userHandler, textHandler, mw)

	router.Setup()

	ctx, cancel := context.WithCancel(context.Background())

	return &Bot{
		bot:    tgBot,
		router: router,
		ctx:    ctx,
		cancel: cancel,
		shed:   shed,
	}, nil
}

func (b *Bot) Start() {
	slog.Info("starting bot polling")

	b.wg.Go(func() {
		<-b.ctx.Done()
		slog.Info("stopping bot polling due to context cancellation")
		b.bot.Stop()
	})

	b.wg.Go(func() {
		b.bot.Start()
	})
}

func (b *Bot) Stop() {
	slog.Info("stop bot polling")

	// Cancel context to signal all goroutines to stop
	b.cancel()

	// Wait for all goroutines to finish
	b.wg.Wait()

	slog.Info("bot polling stopped")

	b.shed.StopAPI()
}

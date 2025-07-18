package bot

import (
	"time"

	bot "support_bot/internal/infra/in/tg"
	"support_bot/internal/infra/in/tg/handlers"
	"support_bot/internal/infra/in/tg/middlewares"
	pgrepo "support_bot/internal/infra/out/pg/repo"
	telegram "support_bot/internal/infra/out/tg"
	"support_bot/internal/service"

	"github.com/jackc/pgx/v5"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	bot    *telebot.Bot
	router *bot.Router
}

func New(
	token string,
	poll time.Duration,
	cleanupTime time.Duration,
	storage *pgx.Conn,
) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poll},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	chatRepo := pgrepo.NewChat(storage)
	userRepo := pgrepo.NewUser(storage)

	chatService := service.NewChat(chatRepo)
	userService := service.NewUser(userRepo)

	messageSender := telegram.NewChatAdaptor(b)

	notifyier := service.NewChatNotify(chatRepo, messageSender)
	userNotifier := service.NewUserNotify(userRepo, messageSender)

	state := handlers.NewState(cleanupTime)

	adminHandler := handlers.NewAdminHandler(
		b,
		userService,
		chatService,
		notifyier,
		userNotifier,
		state,
	)

	userHandler := handlers.NewUserHandler(
		b,
		chatService,
		userService,
		state,
		notifyier,
		userNotifier,
	)

	textHandler := handlers.NewTextHandler(adminHandler, userHandler, state)

	mw := middlewares.NewMw(userService)

	router := bot.NewRouter(b, adminHandler, userHandler, textHandler, mw)

	router.Setup()

	return &Bot{
		bot:    b,
		router: router,
	}, nil
}

func (b *Bot) Start() error {
	b.bot.Start()

	return nil
}

func (b *Bot) Stop() {
	b.bot.Stop()
}

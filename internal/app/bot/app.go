package bot

import (
	"log/slog"
	"support_bot/internal/infra/in/tg/handlers"
	"support_bot/internal/infra/in/tg/middlewares"
	"support_bot/internal/service"
	"time"

	bot "support_bot/internal/infra/in/tg"

	"gopkg.in/telebot.v4"
)

type Bot struct {
	bot    *telebot.Bot
	router *bot.Router
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
	userService *service.User,
	chatService *service.Chat,
	notifyier *service.TelegramNotify,
	statsService *service.Report,
) (*Bot, error) {
	state := handlers.NewState(cleanupTime)

	adminHandler := handlers.NewAdminHandler(
		tgBot,
		userService,
		chatService,
		notifyier,
		statsService,
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

	return &Bot{
		bot:    tgBot,
		router: router,
	}, nil
}

func (b *Bot) Start() {
	slog.Info("starting bot polling")
	go b.bot.Start()
}

func (b *Bot) Stop() {
	slog.Info("stop bot polling")

	b.bot.Stop()
}

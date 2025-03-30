package bot

import (
	"support_bot/internal/bot"
	"support_bot/internal/bot/handlers"
	"support_bot/internal/service"
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	bot    *telebot.Bot
	log    *zap.Logger
	router *bot.Router
}

func New(
	log *zap.Logger,
	token string,
	poll time.Duration,
	sb *service.ServiceBuilder,
) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poll},
	}
	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	router := bot.NewRouter(b, handlers.NewHB(b, sb))

	router.Setup()
	return &Bot{
		bot:    b,
		log:    log,
		router: router,
	}, nil
}

func (b *Bot) Start() error {
	b.log.Info("Bot started")
	b.bot.Start()
	return nil
}

func (b *Bot) Stop() {
	b.bot.Stop()
}

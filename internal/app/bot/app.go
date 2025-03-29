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
	token string,
	poll time.Duration,
	log *zap.Logger,
	us *service.User,
	cs *service.Chat,
	ns *service.Notify,
) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poll},
	}
	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	state := handlers.NewState()
	ah := handlers.NewAdminHandler(
		b,
		us,
		cs,
		ns,
		state,
		log,
	)

	uh := handlers.NewUserHandler(b, cs, us, state, ns)
	th := handlers.NewTextHandler(ah, uh, state)

	router := bot.NewRouter(b, ah, uh, th, us)
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

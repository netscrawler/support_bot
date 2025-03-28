package bot

import (
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	bot *telebot.Bot
	log *zap.Logger
}

func New(
	token string,
	poll time.Duration,
	log *zap.Logger,
) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poll},
	}
	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &Bot{
		bot: b,
		log: log,
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

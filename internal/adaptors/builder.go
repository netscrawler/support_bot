package adaptors

import (
	"support_bot/internal/adaptors/telegram"

	"gopkg.in/telebot.v4"
)

type AdaptorsBuilder struct {
	bot *telebot.Bot
}

func New(bot *telebot.Bot) *AdaptorsBuilder {
	return &AdaptorsBuilder{
		bot: bot,
	}
}

func (ab *AdaptorsBuilder) Build() *telegram.ChatAdaptor {
	return telegram.NewChatAdaptor(ab.bot)
}

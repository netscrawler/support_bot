// Package telegram
package telegram

import (
	"support_bot/internal/models"

	"gopkg.in/telebot.v4"
)

type ChatAdaptor struct{ bot *telebot.Bot }

func NewChatAdaptor(bot *telebot.Bot) *ChatAdaptor {
	return &ChatAdaptor{
		bot: bot,
	}
}

func (ca *ChatAdaptor) Send(chat models.TargetTelegramChat, msg models.TextData) error {
	p := msg.Parse
	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{
		ParseMode: p,
		ThreadID:  chat.ThreadID,
	}
	_, err := ca.bot.Send(c, msg.Msg, o)

	return err
}

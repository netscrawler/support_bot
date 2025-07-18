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

func (ca *ChatAdaptor) Broadcast(
	chats []*telebot.Chat,
	msg string,
	opts ...any,
) (*models.BroadcastResp, error) {
	resp := models.NewBroadcastResp()

	if len(chats) == 0 {
		return nil, models.ErrNotFound
	}

	for _, chat := range chats {
		_, err := ca.bot.Send(chat, msg, opts...)
		if err != nil {
			resp.AddError(chat.Title)

			continue
		}

		resp.AddSuccess()
	}

	return resp, nil
}

func (ca *ChatAdaptor) Send(chat *telebot.Chat, msg string, opts ...any) error {
	_, err := ca.bot.Send(chat, msg, opts...)

	return err
}

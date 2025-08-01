// Package telegram
package telegram

import (
	"bytes"
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

func (ca *ChatAdaptor) SendMedia(chat *telebot.Chat, imgs []*bytes.Buffer, opts ...any) error {
	var album telebot.Album

	for _, img := range imgs {
		photo := &telebot.Photo{
			File: telebot.FromReader(img),
		}

		album = append(album, photo)
	}

	_, err := ca.bot.SendAlbum(chat, album, opts...)

	return err
}

func (ca *ChatAdaptor) SendDocument(
	chat *telebot.Chat,
	buf *bytes.Buffer,
	filename string,
	opts ...any,
) error {
	doc := &telebot.Document{
		File:     telebot.FromReader(buf),
		FileName: filename,
	}

	_, err := ca.bot.Send(chat, doc, opts...)

	return err
}

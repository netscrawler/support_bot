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
	chats []models.Chat,
	msg string,
	opts ...any,
) (*models.BroadcastResp, error) {
	resp := models.NewBroadcastResp()

	if len(chats) == 0 {
		return nil, models.ErrNotFound
	}

	for _, chat := range chats {
		c := &telebot.Chat{ID: chat.ChatID}
		_, err := ca.bot.Send(c, msg, opts...)
		if err != nil {
			resp.AddError(chat.Title)

			continue
		}

		resp.AddSuccess()
	}

	return resp, nil
}

func (ca *ChatAdaptor) Send(chat models.Chat, msg string, opts ...any) error {
	c := &telebot.Chat{ID: chat.ChatID}
	_, err := ca.bot.Send(c, msg, opts...)

	return err
}

func (ca *ChatAdaptor) SendMedia(chat models.Chat, imgs []*bytes.Buffer, opts ...any) error {
	var album telebot.Album
	c := &telebot.Chat{ID: chat.ChatID}

	for _, img := range imgs {
		photo := &telebot.Photo{
			File: telebot.FromReader(img),
		}

		album = append(album, photo)
	}

	_, err := ca.bot.SendAlbum(c, album, opts...)

	return err
}

func (ca *ChatAdaptor) SendDocument(
	chat models.Chat,
	buf *bytes.Buffer,
	filename string,
	opts ...any,
) error {
	doc := &telebot.Document{
		File:     telebot.FromReader(buf),
		FileName: filename,
	}
	c := &telebot.Chat{ID: chat.ChatID}

	_, err := ca.bot.Send(c, doc, opts...)

	return err
}

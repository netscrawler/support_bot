// Package telegram
package telegram

import (
	"errors"
	"strings"

	"support_bot/internal/models"
	"support_bot/internal/pkg"

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
	if err != nil && strings.Contains(err.Error(), "parse") {
		_, err = ca.bot.Send(c, pkg.EscapeMarkdownV2(msg.Msg), o)
	}

	return err
}

func (ca *ChatAdaptor) SendMedia(
	chat models.TargetTelegramChat,
	imgs models.ImageData,
) error {
	var album telebot.Album
	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{ThreadID: chat.ThreadID}

	for img := range imgs.Data() {
		photo := &telebot.Photo{
			File: telebot.FromReader(img),
		}

		album = append(album, photo)
	}

	_, err := ca.bot.SendAlbum(c, album, o)

	return err
}

func (ca *ChatAdaptor) SendDocument(
	chat models.TargetTelegramChat,
	doc models.FileData,
) error {
	o := &telebot.SendOptions{ThreadID: chat.ThreadID}
	c := &telebot.Chat{ID: chat.ChatID}
	var rerr error
	for doc, name := range doc.Data() {
		tgDoc := &telebot.Document{
			File:     telebot.FromReader(doc),
			FileName: name,
		}
		_, err := ca.bot.Send(c, tgDoc, o)
		rerr = errors.Join(rerr, err)

	}

	return rerr
}

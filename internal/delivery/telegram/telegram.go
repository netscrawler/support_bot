// Package telegram
package telegram

import (
	"context"
	"errors"
	"log/slog"

	models "support_bot/internal/models/report"

	"gopkg.in/telebot.v4"
)

type ChatAdaptor struct {
	bot *telebot.Bot
	log *slog.Logger
}

func NewChatAdaptor(bot *telebot.Bot, log *slog.Logger) *ChatAdaptor {
	l := log.With(slog.Any("module", "telegram_sender"))

	return &ChatAdaptor{
		bot: bot,
		log: l,
	}
}

func (ca *ChatAdaptor) Send(
	ctx context.Context,
	chat models.TargetTelegramChat,
	datas ...models.ReportData,
) error {
	if len(datas) == 0 {
		return errors.New("NOTHING TO SEND")
	}

	for _, data := range datas {
		if data == nil {
			ca.log.ErrorContext(ctx, "empty send data", slog.Any("data", data))
		}

		switch data.Kind() {
		case models.SendTextKind:
			dt, ok := data.(*models.TextData)
			if !ok {
				ca.log.ErrorContext(ctx, "invalid telegram send data", slog.Any("data", data))

				continue
			}

			err := ca.sendText(chat, *dt)
			if err != nil {
				ca.log.ErrorContext(ctx, "sending error", slog.Any("error", err))
			}
		case models.SendFileKind:
			dt, ok := data.(*models.FileData)
			if !ok {
				ca.log.ErrorContext(ctx, "invalid telegram send data", slog.Any("data", data))

				continue
			}

			err := ca.sendDocument(chat, *dt)
			if err != nil {
				ca.log.ErrorContext(ctx, "sending error", slog.Any("error", err))
			}
		case models.SendImageKind:
			dt, ok := data.(*models.ImageData)
			if !ok {
				ca.log.ErrorContext(ctx, "invalid telegram send data", slog.Any("data", data))

				continue
			}

			err := ca.sendMedia(chat, *dt)
			if err != nil {
				ca.log.ErrorContext(ctx, "sending error", slog.Any("error", err))
			}
		default:
			ca.log.ErrorContext(ctx, "not supported telegram data", slog.Any("data", data))

			continue
		}
	}

	return nil
}

func (ca *ChatAdaptor) sendText(chat models.TargetTelegramChat, msg models.TextData) error {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.Info("Start sending text message", slog.Any("parse_mode", msg.Parse))
	p := msg.Parse
	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{
		ParseMode: p,
		ThreadID:  chat.ThreadID,
	}

	_, err := ca.bot.Send(c, msg.Msg, o)
	if err != nil {
		l.Error("Error send text message", slog.Any("error", err))

		return err
	}

	l.Info("Successfully send text message")

	return nil
}

func (ca *ChatAdaptor) sendMedia(
	chat models.TargetTelegramChat,
	imgs models.ImageData,
) error {
	var album telebot.Album

	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.Info("Start sending media")

	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{ThreadID: chat.ThreadID}

	for img := range imgs.Data() {
		photo := &telebot.Photo{
			File: telebot.FromReader(img),
		}

		album = append(album, photo)
	}

	_, err := ca.bot.SendAlbum(c, album, o)
	if err != nil {
		l.Error("Error send media", slog.Any("error", err))

		return err
	}

	l.Info("Successfully send media")

	return nil
}

func (ca *ChatAdaptor) sendDocument(
	chat models.TargetTelegramChat,
	doc models.FileData,
) error {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.Info("Start sending document")

	o := &telebot.SendOptions{ThreadID: chat.ThreadID}
	c := &telebot.Chat{ID: chat.ChatID}

	var rerr error

	for doc, name := range doc.Data() {
		tgDoc := &telebot.Document{
			File:     telebot.FromReader(doc),
			FileName: name,
		}

		_, err := ca.bot.Send(c, tgDoc, o)
		if err != nil {
			l.Error(
				"Error send document",
				slog.Any("error", err),
				slog.Any("document_name", tgDoc.FileName),
			)
			rerr = errors.Join(rerr, err)

			continue
		}

		l.Info("Successfully send document", slog.Any("document_name", tgDoc.FileName))
	}

	return rerr
}

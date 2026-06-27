// Package telegram
package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
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

func (ca *ChatAdaptor) SendText(
	ctx context.Context,
	chat models.TgChat,
	msg string,
) (*models.TgMessage, error) {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending text message")
	p := telebot.ModeHTML
	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{
		ParseMode: p,
		ThreadID:  chat.ThreadID,
	}

	tgMsg, err := ca.bot.Send(c, msg, o)
	if err != nil {
		return nil, fmt.Errorf("error send text message: %w", err)
	}

	return models.NewFromTelebot(tgMsg), nil
}

func (ca *ChatAdaptor) SendMedia(
	ctx context.Context,
	chat models.TgChat,
	imgs []models.Data,
) ([]models.TgMessage, error) {
	var album telebot.Album

	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending media")

	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{ThreadID: chat.ThreadID}

	for _, i := range imgs {
		photo := &telebot.Photo{
			File:    telebot.FromReader(i.Data),
			Caption: i.Name,
		}

		album = append(album, photo)
	}

	tgMsg, err := ca.bot.SendAlbum(c, album, o)
	if err != nil {
		l.ErrorContext(ctx, "Error send media", slog.Any("error", err))

		return nil, err
	}

	l.InfoContext(ctx, "Successfully send media")

	return models.NewMsgFromTelebotMany(tgMsg), nil
}

func (ca *ChatAdaptor) SendDocument(
	ctx context.Context,
	chat models.TgChat,
	doc []models.Data,
) ([]models.TgMessage, error) {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending document")

	o := &telebot.SendOptions{ThreadID: chat.ThreadID}
	c := &telebot.Chat{ID: chat.ChatID}

	var retErr error

	var retMsg []models.TgMessage

	for _, f := range doc {
		doc, name := f.Data, f.Name
		tgDoc := &telebot.Document{
			File:     telebot.FromReader(doc),
			FileName: name,
		}

		tgMsg, err := ca.bot.Send(c, tgDoc, o)
		if err != nil {
			l.ErrorContext(
				ctx,
				"Error send document",
				slog.Any("error", err),
				slog.Any("document_name", tgDoc.FileName),
			)
			retErr = errors.Join(retErr, err)

			continue
		}
		retMsg = append(retMsg, *models.NewFromTelebot(tgMsg))

		l.InfoContext(ctx, "Successfully send document", slog.Any("document_name", tgDoc.FileName))
	}

	return retMsg, retErr
}

func (ca *ChatAdaptor) DeleteMsg(message models.TgMessage) error {
	return ca.bot.Delete(telebot.StoredMessage{
		MessageID: strconv.Itoa(message.MessageID),
		ChatID:    message.ChatID,
	})
}

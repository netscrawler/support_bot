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
	datas ...models.ExportedReport,
) error {
	if len(datas) == 0 {
		return errors.New("NOTHING TO SEND")
	}

	var files []models.FileData
	var imgs []models.FileData

	for _, data := range datas {
		switch data.Type {
		case models.TextType:
			txt, err := data.ToTextData()
			if err != nil {
				return err
			}

			err = ca.sendText(ctx, chat, *txt)
			if err != nil {
				ca.log.ErrorContext(ctx, "sending error", slog.Any("error", err))
			}
		case models.SendFileKind:
			file, err := data.ToFileData()
			if err != nil {
				return err
			}

			files = append(files, *file)
		case models.SendImageKind:
			img, err := data.ToFileData()
			if err != nil {
				return err
			}

			imgs = append(imgs, *img)

		default:
			ca.log.ErrorContext(ctx, "not supported telegram data", slog.Any("data", data))

			continue
		}
	}

	var fileErr, imgErr error
	if len(files) > 0 {
		fileErr = ca.sendDocument(ctx, chat, files...)
	}
	if len(imgs) > 0 {
		fileErr = ca.sendMedia(ctx, chat, imgs...)
	}

	return errors.Join(fileErr, imgErr)
}

func (ca *ChatAdaptor) sendText(
	ctx context.Context,
	chat models.TargetTelegramChat,
	msg models.TextData,
) error {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending text message", slog.Any("parse_mode", msg.Parse))
	p := msg.Parse
	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{
		ParseMode: p,
		ThreadID:  chat.ThreadID,
	}

	_, err := ca.bot.Send(c, msg.Msg, o)
	if err != nil {
		l.ErrorContext(ctx, "Error send text message", slog.Any("error", err))

		return err
	}

	l.InfoContext(ctx, "Successfully send text message")

	return nil
}

func (ca *ChatAdaptor) sendMedia(
	ctx context.Context,
	chat models.TargetTelegramChat,
	imgs ...models.FileData,
) error {
	var album telebot.Album

	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending media")

	c := &telebot.Chat{ID: chat.ChatID}
	o := &telebot.SendOptions{ThreadID: chat.ThreadID}

	for _, img := range imgs {
		photo := &telebot.Photo{
			File: telebot.FromReader(img.File),
		}

		album = append(album, photo)
	}

	_, err := ca.bot.SendAlbum(c, album, o)
	if err != nil {
		l.ErrorContext(ctx, "Error send media", slog.Any("error", err))

		return err
	}

	l.InfoContext(ctx, "Successfully send media")

	return nil
}

func (ca *ChatAdaptor) sendDocument(
	ctx context.Context,
	chat models.TargetTelegramChat,
	docs ...models.FileData,
) error {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending document")

	o := &telebot.SendOptions{ThreadID: chat.ThreadID}
	c := &telebot.Chat{ID: chat.ChatID}

	var rerr error

	for _, doc := range docs {
		tgDoc := &telebot.Document{
			File:     telebot.FromReader(doc.File),
			FileName: doc.Name,
		}

		_, err := ca.bot.Send(c, tgDoc, o)
		if err != nil {
			l.ErrorContext(
				ctx,
				"Error send document",
				slog.Any("error", err),
				slog.Any("document_name", tgDoc.FileName),
			)
			rerr = errors.Join(rerr, err)

			continue
		}

		l.InfoContext(ctx, "Successfully send document", slog.Any("document_name", tgDoc.FileName))
	}

	return rerr
}

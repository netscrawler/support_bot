// Package telegram
package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"support_bot/internal/models"
	"support_bot/internal/pkg/retry"

	"golang.org/x/time/rate"
	"gopkg.in/telebot.v4"
)

type ChatAdaptor struct {
	bot *telebot.Bot
	log *slog.Logger
	rl  *rate.Limiter

	retry *retry.Retry
}

func NewChatAdaptor(bot *telebot.Bot, retry *retry.Retry, log *slog.Logger) *ChatAdaptor {
	l := log.With(slog.Any("module", "telegram_sender"))
	rl := rate.NewLimiter(rate.Limit(9), 5)

	retry.AddRateLimit(rl)

	return &ChatAdaptor{
		bot: bot,
		rl:  rl,

		log:   l,
		retry: retry,
	}
}

func (ca *ChatAdaptor) SendText(
	ctx context.Context,
	chat models.TgChat,
	msg string,
) (*models.SentMessage, error) {
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

	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}

	tgMsg, err := ca.bot.Send(c, msg, o)
	if err != nil {
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("%d", chat.ChatID), func(_ context.Context) error {
				_, err := ca.bot.Send(c, msg, o)
				return err
			}),
		)
		if retryErr != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error send text message: %w", err)
	}

	return models.NewFromTelebot(tgMsg), nil
}

func (ca *ChatAdaptor) SendMedia(
	ctx context.Context,
	chat models.TgChat,
	imgs []models.Data,
) ([]models.SentMessage, error) {
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

	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}
	tgMsg, err := ca.bot.SendAlbum(c, album, o)
	if err != nil {
		l.ErrorContext(ctx, "Error send media", slog.Any("error", err))
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("%d", chat.ChatID), func(_ context.Context) error {
				_, err := ca.bot.SendAlbum(c, album, o)
				return err
			}),
		)
		if retryErr != nil {
			return nil, err
		}

		return nil, err
	}

	l.InfoContext(ctx, "Successfully send media")

	return models.NewMsgFromTelebotMany(tgMsg), nil
}

func (ca *ChatAdaptor) SendDocument(
	ctx context.Context,
	chat models.TgChat,
	doc []models.Data,
) ([]models.SentMessage, error) {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending document")

	o := &telebot.SendOptions{ThreadID: chat.ThreadID}
	c := &telebot.Chat{ID: chat.ChatID}

	var retErr error

	var retMsg []models.SentMessage

	for _, f := range doc {
		doc, name := f.Data, f.Name
		tgDoc := &telebot.Document{
			File:     telebot.FromReader(doc),
			FileName: name,
		}

		if err := ca.rl.Wait(ctx); err != nil {
			return nil, err
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

			ca.retry.Submit(
				retry.NewTask(fmt.Sprintf("%d", chat.ChatID), func(_ context.Context) error {
					_, err := ca.bot.Send(c, tgDoc, o)
					return err
				}),
			)

			continue
		}

		retMsg = append(retMsg, *models.NewFromTelebot(tgMsg))

		l.InfoContext(ctx, "Successfully send document", slog.Any("document_name", tgDoc.FileName))
	}

	return retMsg, retErr
}

func (ca *ChatAdaptor) DeleteMsg(ctx context.Context, message models.SentMessage) error {
	if err := ca.rl.Wait(ctx); err != nil {
		return err
	}

	err := ca.bot.Delete(telebot.StoredMessage{
		MessageID: strconv.Itoa(message.MessageID),
		ChatID:    message.ChatID,
	})
	if err != nil {
		ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("%d", message.ID), func(_ context.Context) error {
				err := ca.bot.Delete(telebot.StoredMessage{
					MessageID: strconv.Itoa(message.MessageID),
					ChatID:    message.ChatID,
				})

				return err
			}),
		)
	}
	return err
}

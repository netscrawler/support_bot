// Package telegram реализует доставку отчётов в Telegram.
package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"support_bot/internal/models"
	"support_bot/internal/pkg/retry"

	"github.com/mymmrac/telego"
	"golang.org/x/time/rate"
)

type ChatAdaptor struct {
	bot *telego.Bot
	log *slog.Logger
	rl  *rate.Limiter

	delays retry.Delays
}

func NewChatAdaptor(
	bot *telego.Bot,
	delays retry.Delays,
	log *slog.Logger,
) *ChatAdaptor {
	l := log.With(slog.Any("module", "telegram_sender"))

	return &ChatAdaptor{
		bot: bot,
		rl:  rate.NewLimiter(rate.Limit(9), 5),

		log:    l,
		delays: delays,
	}
}

// reader возвращает функцию, отдающую на каждый вызов свежую копию d:
// bytes.Buffer опустошается при отправке, и на повторе файл ушёл бы пустым.
func reader(d models.Data) func() models.Data {
	raw := d.Data.Bytes()

	return func() models.Data {
		d.Data = bytes.NewBuffer(raw)

		return d
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

	var tgMsg *telego.Message

	err := ca.do(
		ctx,
		"telegram.SendText",
		func(ctx context.Context) (err error) {
			tgMsg, err = ca.bot.SendMessage(ctx, &telego.SendMessageParams{
				ChatID:          telego.ChatID{ID: chat.ChatID},
				MessageThreadID: chat.ThreadID,
				Text:            msg,
				ParseMode:       telego.ModeHTML,
			})

			return err
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error send text message: %w", err)
	}

	return models.NewFromTelego(tgMsg), nil
}

func (ca *ChatAdaptor) SendRichText(
	ctx context.Context,
	chat models.TgChat,
	msg string,
) (*models.SentMessage, error) {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending rich text message")

	var tgMsg *telego.Message

	err := ca.do(
		ctx,
		"telegram.SendRichText",
		func(ctx context.Context) (err error) {
			tgMsg, err = ca.bot.SendRichMessage(
				ctx,
				&telego.SendRichMessageParams{
					ChatID:          telego.ChatID{ID: chat.ChatID},
					MessageThreadID: chat.ThreadID,
					RichMessage:     telego.InputRichMessage{HTML: msg},
				},
			)

			return err
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error send rich text message: %w", err)
	}

	return models.NewFromTelego(tgMsg), nil
}

func (ca *ChatAdaptor) SendMedia(
	ctx context.Context,
	chat models.TgChat,
	imgs []models.Data,
) ([]models.SentMessage, error) {
	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending media")

	fresh := make([]func() models.Data, len(imgs))
	for i, img := range imgs {
		fresh[i] = reader(img)
	}

	var tgMsg []telego.Message

	err := ca.do(
		ctx,
		"telegram.SendMedia",
		func(ctx context.Context) (err error) {
			album := make([]telego.InputMedia, 0, len(fresh))
			for _, f := range fresh {
				album = append(album, &telego.InputMediaPhoto{
					Type:  telego.MediaTypePhoto,
					Media: telego.InputFile{File: f()},
				})
			}

			tgMsg, err = ca.bot.SendMediaGroup(
				ctx,
				&telego.SendMediaGroupParams{
					ChatID:          telego.ChatID{ID: chat.ChatID},
					MessageThreadID: chat.ThreadID,
					Media:           album,
				},
			)

			return err
		},
	)
	if err != nil {
		l.ErrorContext(ctx, "Error send media", slog.Any("error", err))

		return nil, err
	}

	l.InfoContext(ctx, "Successfully send media")

	return models.NewFromTelegoMany(tgMsg), nil
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

	var retErr error

	var retMsg []models.SentMessage

	for _, f := range doc {
		fresh := reader(f)

		var tgMsg *telego.Message

		err := ca.do(
			ctx,
			"telegram.SendDocument",
			func(ctx context.Context) (err error) {
				tgMsg, err = ca.bot.SendDocument(
					ctx,
					&telego.SendDocumentParams{
						ChatID:          telego.ChatID{ID: chat.ChatID},
						MessageThreadID: chat.ThreadID,
						Document:        telego.InputFile{File: fresh()},
					},
				)

				return err
			},
		)
		if err != nil {
			l.ErrorContext(
				ctx,
				"Error send document",
				slog.Any("error", err),
				slog.Any("document_name", f.FileName),
			)

			retErr = errors.Join(retErr, err)

			continue
		}

		retMsg = append(retMsg, *models.NewFromTelego(tgMsg))

		l.InfoContext(
			ctx,
			"Successfully send document",
			slog.Any("document_name", f.FileName),
		)
	}

	return retMsg, retErr
}

func (ca *ChatAdaptor) DeleteMsg(
	ctx context.Context,
	message models.SentMessage,
) error {
	return ca.do(ctx, "telegram.DeleteMsg", func(ctx context.Context) error {
		return ca.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
			MessageID: message.MessageID,
			ChatID:    telego.ChatID{ID: message.ChatID},
		})
	})
}

// do прогоняет вызов API через retry.Do; лимитер ждётся на каждую попытку.
func (ca *ChatAdaptor) do(
	ctx context.Context,
	op string,
	call func(context.Context) error,
) error {
	return retry.Do(
		ctx,
		ca.delays,
		ca.log,
		op,
		func(ctx context.Context) error {
			if err := ca.rl.Wait(ctx); err != nil {
				return retry.Permanent(err)
			}

			return classify(call(ctx))
		},
	)
}

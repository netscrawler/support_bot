// Package telegram
package telegram

import (
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

	retry *retry.Retry
}

func NewChatAdaptor(bot *telego.Bot, retry *retry.Retry, log *slog.Logger) *ChatAdaptor {
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

	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}

	tgMsg, err := ca.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{
			ID: chat.ChatID,
		},
		MessageThreadID: chat.ThreadID,
		Text:            msg,
		ParseMode:       telego.ModeHTML,
	})
	if err != nil {
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("%d", chat.ChatID), func(ctx context.Context) error {
				_, err := ca.bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{
						ID: chat.ChatID,
					},
					MessageThreadID: chat.ThreadID,
					Text:            msg,
					ParseMode:       telego.ModeHTML,
				})
				return err
			}),
		)
		if retryErr != nil {
			return nil, err
		}
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

	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}

	tgMsg, err := ca.bot.SendRichMessage(ctx, &telego.SendRichMessageParams{
		ChatID: telego.ChatID{
			ID: chat.ChatID,
		},
		MessageThreadID: chat.ThreadID,
		RichMessage:     telego.InputRichMessage{HTML: msg},
	})
	if err != nil {
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("%d", chat.ChatID), func(ctx context.Context) error {
				_, err := ca.bot.SendRichMessage(ctx, &telego.SendRichMessageParams{
					ChatID: telego.ChatID{
						ID: chat.ChatID,
					},
					MessageThreadID: chat.ThreadID,
					RichMessage:     telego.InputRichMessage{HTML: msg},
				})
				return err
			}),
		)
		if retryErr != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error send text message: %w", err)
	}

	return models.NewFromTelego(tgMsg), nil
}

func (ca *ChatAdaptor) SendMedia(
	ctx context.Context,
	chat models.TgChat,
	imgs []models.Data,
) ([]models.SentMessage, error) {
	var album []telego.InputMedia

	l := ca.log.With(
		slog.Group(
			"recipient",
			slog.Any("chat", chat.ChatID), slog.Any("thread id", chat.ThreadID),
		))

	l.InfoContext(ctx, "Start sending media")

	for _, i := range imgs {
		photo := telego.InputMediaPhoto{
			Type: telego.MediaTypePhoto,
			Media: telego.InputFile{
				File: &i,
			},
		}

		album = append(album, &photo)
	}

	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}

	tgMsg, err := ca.bot.SendMediaGroup(ctx, &telego.SendMediaGroupParams{
		ChatID:          telego.ChatID{ID: chat.ChatID},
		MessageThreadID: chat.ThreadID,
		Media:           album,
	})
	if err != nil {
		l.ErrorContext(ctx, "Error send media", slog.Any("error", err))
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("%d", chat.ChatID), func(_ context.Context) error {
				_, err := ca.bot.SendMediaGroup(ctx, &telego.SendMediaGroupParams{
					ChatID:          telego.ChatID{ID: chat.ChatID},
					MessageThreadID: chat.ThreadID,
					Media:           album,
				})
				return err
			}),
		)
		if retryErr != nil {
			return nil, err
		}

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

		if err := ca.rl.Wait(ctx); err != nil {
			return nil, err
		}

		tgMsg, err := ca.bot.SendDocument(ctx, &telego.SendDocumentParams{
			ChatID: telego.ChatID{
				ID:       chat.ChatID,
				Username: "",
			},
			MessageThreadID: chat.ThreadID,
			Document: telego.InputFile{
				File: f,
			},
		})
		if err != nil {
			l.ErrorContext(
				ctx,
				"Error send document",
				slog.Any("error", err),
				slog.Any("document_name", f.FileName),
			)
			retErr = errors.Join(retErr, err)

			ca.retry.Submit(
				retry.NewTask(fmt.Sprintf("%d", chat.ChatID), func(_ context.Context) error {
					_, err := ca.bot.SendDocument(ctx, &telego.SendDocumentParams{
						ChatID: telego.ChatID{
							ID:       chat.ChatID,
							Username: "",
						},
						MessageThreadID: chat.ThreadID,
						Document: telego.InputFile{
							File: f,
						},
					})
					return err
				}),
			)

			continue
		}

		retMsg = append(retMsg, *models.NewFromTelego(tgMsg))

		l.InfoContext(ctx, "Successfully send document", slog.Any("document_name", f.FileName))
	}

	return retMsg, retErr
}

func (ca *ChatAdaptor) DeleteMsg(ctx context.Context, message models.SentMessage) error {
	if err := ca.rl.Wait(ctx); err != nil {
		return err
	}

	err := ca.bot.DeleteMessage(
		ctx,
		&telego.DeleteMessageParams{
			MessageID: message.MessageID,
			ChatID:    telego.ChatID{ID: message.ChatID},
		},
	)
	if err != nil {
		ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("%d", message.ID), func(_ context.Context) error {
				err := ca.bot.DeleteMessage(
					ctx,
					&telego.DeleteMessageParams{
						MessageID: message.MessageID,
						ChatID:    telego.ChatID{ID: message.ChatID},
					},
				)

				return err
			}),
		)
	}
	return err
}

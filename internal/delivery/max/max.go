package maxadp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"support_bot/internal/models"
	"support_bot/internal/pkg/retry"

	maxcli "github.com/max-messenger/max-bot-api-client-go/v2"
	maxModel "github.com/max-messenger/max-bot-api-client-go/v2/model"
	"golang.org/x/time/rate"
)

var ErrMaxAdaptorInactive = errors.New("max adaptor inactive from config")

type Adaptor struct {
	api *maxcli.Api

	rl *rate.Limiter

	retry *retry.Retry

	active bool

	log *slog.Logger
}

func New(api *maxcli.Api, retr *retry.Retry, active bool, log *slog.Logger) *Adaptor {
	l := log.With(slog.Any("module", "max_sender"))
	adp := &Adaptor{
		log:    l,
		active: active,
	}

	if !active {
		return adp
	}

	adp.api = api
	adp.rl = rate.NewLimiter(rate.Limit(9), 5)
	retr.AddRateLimit(adp.rl)
	adp.retry = retr

	return adp
}

func (ca *Adaptor) SendText(
	ctx context.Context,
	chat models.MaxChat,
	msg string,
) (*models.SentMessage, error) {
	if !ca.active {
		return nil, ErrMaxAdaptorInactive
	}
	maxMsg := &maxcli.Message{}
	maxMsg = maxMsg.SetText(msg).SetChat(chat.ChatID).SetFormat(maxModel.FormatHTML)
	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}

	repl, err := ca.api.Messages.Send(ctx, maxMsg)
	if err != nil {
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("max %d", chat.ChatID), func(ctx context.Context) error {
				_, err := ca.api.Messages.Send(ctx, maxMsg)
				return err
			}),
		)
		if retryErr != nil {
			return nil, errors.Join(err, retryErr)
		}
	}

	replMsg := models.NewMsgFromMax(repl.Message)

	return replMsg, nil
}

func (ca *Adaptor) SendMedia(
	ctx context.Context,
	chat models.MaxChat,
	imgs []models.Data,
) (*models.SentMessage, error) {
	if !ca.active {
		return nil, ErrMaxAdaptorInactive
	}
	maxMsg := &maxcli.Message{}
	maxMsg.SetChat(chat.ChatID)

	for _, i := range imgs {
		if err := ca.rl.Wait(ctx); err != nil {
			return nil, err
		}

		token, err := ca.api.Upload.Upload(
			ctx,
			maxModel.UploadImage,
			i.Data,
			i.Name,
			int64(i.Data.Len()),
		)
		if err != nil {
			continue
		}

		maxMsg.AddAttachByToken(token, maxModel.AttachImage)

	}

	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}

	repl, err := ca.api.Messages.Send(ctx, maxMsg)
	if err != nil {
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("max %d", chat.ChatID), func(ctx context.Context) error {
				_, err := ca.api.Messages.Send(ctx, maxMsg)
				return err
			}),
		)
		if retryErr != nil {
			return nil, errors.Join(err, retryErr)
		}
	}

	replMsg := models.NewMsgFromMax(repl.Message)

	return replMsg, nil
}

func (ca *Adaptor) SendDocument(
	ctx context.Context,
	chat models.MaxChat,
	doc []models.Data,
) (*models.SentMessage, error) {
	if !ca.active {
		return nil, ErrMaxAdaptorInactive
	}
	maxMsg := &maxcli.Message{}
	maxMsg.SetChat(chat.ChatID)

	for _, i := range doc {
		if err := ca.rl.Wait(ctx); err != nil {
			return nil, err
		}

		token, err := ca.api.Upload.Upload(
			ctx,
			maxModel.UploadFile,
			i.Data,
			i.Name,
			int64(i.Data.Len()),
		)
		if err != nil {
			continue
		}

		maxMsg.AddAttachByToken(token, maxModel.AttachImage)

	}

	if err := ca.rl.Wait(ctx); err != nil {
		return nil, err
	}

	repl, err := ca.api.Messages.Send(ctx, maxMsg)
	if err != nil {
		retryErr := ca.retry.Submit(
			retry.NewTask(fmt.Sprintf("max %d", chat.ChatID), func(ctx context.Context) error {
				_, err := ca.api.Messages.Send(ctx, maxMsg)
				return err
			}),
		)
		if retryErr != nil {
			return nil, errors.Join(err, retryErr)
		}
	}

	replMsg := models.NewMsgFromMax(repl.Message)

	return replMsg, nil
}

func (ca *Adaptor) DeleteMsg(ctx context.Context, message models.SentMessage) error {
	if !ca.active {
		return ErrMaxAdaptorInactive
	}
	if err := ca.rl.Wait(context.Background()); err != nil {
		return err
	}

	r, err := ca.api.Messages.DeleteMessage(ctx, message.MessageIDStr)
	if err != nil {
		retryErr := ca.retry.Submit(
			retry.NewTask(
				fmt.Sprintf("max %d", message.MessageID),
				func(ctx context.Context) error {
					_, err := ca.api.Messages.DeleteMessage(ctx, message.MessageIDStr)

					return err
				},
			),
		)
		if retryErr != nil {
			return errors.Join(err, retryErr)
		}

		return err
	}

	if !r.Success {
		return errors.New("failed to delete message")
	}

	return nil
}

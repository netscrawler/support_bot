package maxadp

import (
	"bytes"
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

	delays retry.Delays

	active bool

	log *slog.Logger
}

func New(
	api *maxcli.Api,
	delays retry.Delays,
	active bool,
	log *slog.Logger,
) *Adaptor {
	l := log.With(slog.Any("module", "max_sender"))
	adp := &Adaptor{
		log:    l,
		active: active,
		delays: delays,
	}

	if !active {
		return adp
	}

	adp.api = api
	adp.rl = rate.NewLimiter(rate.Limit(9), 5)

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

	maxMsg = maxMsg.SetText(msg).
		SetChat(chat.ChatID).
		SetFormat(maxModel.FormatHTML)

	var repl maxModel.SendMessageResult

	err := ca.do(ctx, "max.Send", func(ctx context.Context) (err error) {
		repl, err = ca.api.Messages.Send(ctx, maxMsg)

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("max send to chat %d: %w", chat.ChatID, err)
	}

	replMsg := models.NewMsgFromMax(repl.Message)

	return replMsg, nil
}

func (ca *Adaptor) SendMedia(
	ctx context.Context,
	chat models.MaxChat,
	imgs []models.Data,
) (*models.SentMessage, error) {
	return ca.sendUploads(ctx, chat, maxModel.UploadImage, imgs)
}

func (ca *Adaptor) SendDocument(
	ctx context.Context,
	chat models.MaxChat,
	doc []models.Data,
) (*models.SentMessage, error) {
	return ca.sendUploads(ctx, chat, maxModel.UploadFile, doc)
}

func (ca *Adaptor) sendUploads(
	ctx context.Context,
	chat models.MaxChat,
	kind maxModel.UploadType,
	files []models.Data,
) (*models.SentMessage, error) {
	if !ca.active {
		return nil, ErrMaxAdaptorInactive
	}

	maxMsg := &maxcli.Message{}
	maxMsg.SetChat(chat.ChatID)

	attach := maxModel.AttachFile
	if kind == maxModel.UploadImage {
		attach = maxModel.AttachImage
	}

	var uploadErr error

	tokens := 0

	for _, i := range files {
		token, err := ca.upload(ctx, kind, i)
		if err != nil {
			uploadErr = errors.Join(
				uploadErr,
				fmt.Errorf("upload %s: %w", i.FileName, err),
			)

			continue
		}

		maxMsg.AddAttachByToken(token, attach)

		tokens++
	}

	if tokens == 0 {
		return nil, uploadErr
	}

	var repl maxModel.SendMessageResult

	err := ca.do(ctx, "max.Send", func(ctx context.Context) (err error) {
		repl, err = ca.api.Messages.Send(ctx, maxMsg)

		return err
	})
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("max send to chat %d: %w", chat.ChatID, err),
			uploadErr,
		)
	}

	return models.NewMsgFromMax(repl.Message), uploadErr
}

func (ca *Adaptor) DeleteMsg(
	ctx context.Context,
	message models.SentMessage,
) error {
	if !ca.active {
		return ErrMaxAdaptorInactive
	}

	var r maxModel.SimpleQueryResult

	err := ca.do(ctx, "max.DeleteMsg", func(ctx context.Context) (err error) {
		r, err = ca.api.Messages.DeleteMessage(ctx, deRef(message.MessageIDStr))

		return err
	})
	if err != nil {
		return fmt.Errorf("max delete message: %w", err)
	}

	if !r.Success {
		return errors.New("failed to delete message")
	}

	return nil
}

// do ждёт лимитер на каждой попытке; ошибка лимитера (отмена ctx) не повторяется.
func (ca *Adaptor) do(
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

			return call(ctx)
		},
	)
}

// upload грузит файл; bytes.Buffer опустошается при чтении, поэтому на каждую попытку свежий reader.
func (ca *Adaptor) upload(
	ctx context.Context,
	typ maxModel.UploadType,
	d models.Data,
) (string, error) {
	raw := d.Data.Bytes()

	var token string

	err := ca.do(ctx, "max.Upload", func(ctx context.Context) (err error) {
		token, err = ca.api.Upload.Upload(
			ctx,
			typ,
			bytes.NewReader(raw),
			d.FileName,
			int64(len(raw)),
		)

		return err
	})

	return token, err
}

func deRef[T any](t *T) T {
	var tt T

	if t == nil {
		return tt
	}

	return *t
}

package delivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"support_bot/internal/delivery/smtp"
	"support_bot/internal/metrics"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"
)

var (
	errEmptyRecipient       = errors.New("empty recipient")
	errInvalidChatRecipient = errors.New("invalid recipient")
)

type TgSender interface {
	SendText(
		ctx context.Context,
		rcpt models.TgChat,
		s string,
	) (*models.SentMessage, error)
	SendRichText(
		ctx context.Context,
		rcpt models.TgChat,
		s string,
	) (*models.SentMessage, error)
	SendDocument(
		ctx context.Context,
		rcpt models.TgChat,
		model []models.Data,
	) ([]models.SentMessage, error)
	SendMedia(
		ctx context.Context,
		rcpt models.TgChat,
		model []models.Data,
	) ([]models.SentMessage, error)
}

type SmbSender interface {
	Upload(ctx context.Context, remote string, data models.Data) error
}

type SMTPSender interface {
	Send(ctx context.Context, mail smtp.Mail) error
}

type MaxSender interface {
	SendText(
		ctx context.Context,
		rcpt models.MaxChat,
		s string,
	) (*models.SentMessage, error)
	SendDocument(
		ctx context.Context,
		rcpt models.MaxChat,
		model []models.Data,
	) (*models.SentMessage, error)
	SendMedia(
		ctx context.Context,
		rcpt models.MaxChat,
		model []models.Data,
	) (*models.SentMessage, error)
}

type Dispatcher struct {
	tg   TgSender
	smb  SmbSender
	smtp SMTPSender
	max  MaxSender
}

func NewDispatcher(
	tg TgSender,
	smb SmbSender,
	smtpSender SMTPSender,
	maxSender MaxSender,
) *Dispatcher {
	return &Dispatcher{tg, smb, smtpSender, maxSender}
}

// TgText отправляет служебный текст в Telegram-чат.
func (d *Dispatcher) TgText(
	ctx context.Context,
	chatID int64,
	msg string,
) error {
	_, err := d.tg.SendText(ctx, models.TgChat{ChatID: chatID}, msg)

	return err
}

func (d *Dispatcher) Send(ctx context.Context, m *models.Message) (
	sentMsg []models.SentMessage, delivered int, sendErr error,
) {
	for _, r := range m.Recipients {
		var (
			err     error
			channel string
		)

		switch r.Type {
		case models.TelegramRecipient:
			channel = "tg"

			var msg []models.SentMessage

			msg, err = sendTg(ctx, m, d.tg, r)
			if err == nil && r.NeedDeleteAfterEndOfDay {
				sentMsg = append(sentMsg, msg...)
			}

		case models.EmailRecipient:
			channel = "smtp"
			err = sendSMTP(ctx, m, d.smtp, r)

		case models.SambaRecipient:
			channel = "smb"
			err = sendSMB(ctx, m, d.smb, r)

		case models.MaxRecipient:
			channel = "max"

			var msg []models.SentMessage

			msg, err = sendMax(ctx, m, d.max, r)
			// частично отправленное сообщение нужно удалить в конце дня даже при ошибке
			if r.NeedDeleteAfterEndOfDay {
				sentMsg = append(sentMsg, msg...)
			}

		default:
			channel = "unknown"
			err = fmt.Errorf("unsupported recipient type: %s", r.Type)
		}

		status := "ok"
		if err != nil {
			status = "failed"
			sendErr = errors.Join(sendErr, err)
		} else {
			delivered++
		}

		metrics.DeliveryRecipient.WithLabelValues(channel, status, metrics.Reason(err)).
			Inc()
	}

	return sentMsg, delivered, sendErr
}

func sendTg(
	ctx context.Context,
	m *models.Message,
	sender TgSender,
	r models.Recipient,
) ([]models.SentMessage, error) {
	if r.Chat == nil {
		return nil, errEmptyRecipient
	}

	if r.Chat.ChType != models.ChatTypeTg {
		return nil, fmt.Errorf(
			"%w: %s for recipient %s",
			errInvalidChatRecipient,
			r.Chat.ChType,
			r.Type,
		)
	}

	rcpt := models.TgChat{ChatID: r.Chat.ChatID, ThreadID: deRef(r.ThreadID, 0)}

	var sendErr error

	var retMsg []models.SentMessage

	if m.Text != nil {
		for _, data := range m.Text {
			msg, err := sender.SendText(ctx, rcpt, data.Data.String())
			if err != nil {
				sendErr = fmt.Errorf("sending text: %w", err)
			} else {
				retMsg = append(retMsg, *msg)
			}
		}
	}

	if m.RichText != nil {
		for _, data := range m.RichText {
			msg, err := sender.SendRichText(ctx, rcpt, data.Data.String())
			if err != nil {
				sendErr = fmt.Errorf("sending rich text: %w", err)
			} else {
				retMsg = append(retMsg, *msg)
			}
		}
	}

	if m.Files != nil {
		msg, err := sender.SendDocument(ctx, rcpt, m.Files)
		if err != nil {
			sendErr = fmt.Errorf("sending document: %w", err)
		} else {
			retMsg = append(retMsg, msg...)
		}
	}

	if m.Images != nil {
		msg, err := sender.SendMedia(ctx, rcpt, m.Images)
		if err != nil {
			sendErr = fmt.Errorf("sending document: %w", err)
		} else {
			retMsg = append(retMsg, msg...)
		}
	}

	return retMsg, sendErr
}

func sendMax(
	ctx context.Context,
	m *models.Message,
	sender MaxSender,
	r models.Recipient,
) ([]models.SentMessage, error) {
	if r.Chat == nil {
		return nil, errEmptyRecipient
	}

	if r.Chat.ChType != models.ChatTypeMax {
		return nil, fmt.Errorf(
			"%w: %s for recipient %s",
			errInvalidChatRecipient,
			r.Chat.ChType,
			r.Type,
		)
	}

	rcpt := models.MaxChat{ChatID: r.Chat.ChatID}

	var sendErr error

	var retMsg []models.SentMessage

	if m.Text != nil {
		for _, data := range m.Text {
			msg, err := sender.SendText(ctx, rcpt, data.Data.String())
			if err != nil {
				sendErr = errors.Join(
					sendErr,
					fmt.Errorf("sending text: %w", err),
				)
			}

			if msg != nil {
				retMsg = append(retMsg, *msg)
			}
		}
	}

	if m.Files != nil {
		msg, err := sender.SendDocument(ctx, rcpt, m.Files)
		if err != nil {
			sendErr = errors.Join(
				sendErr,
				fmt.Errorf("sending document: %w", err),
			)
		}

		if msg != nil {
			retMsg = append(retMsg, *msg)
		}
	}

	if m.Images != nil {
		msg, err := sender.SendMedia(ctx, rcpt, m.Images)
		if err != nil {
			sendErr = errors.Join(sendErr, fmt.Errorf("sending media: %w", err))
		}

		if msg != nil {
			retMsg = append(retMsg, *msg)
		}
	}

	return retMsg, sendErr
}

func sendSMB(
	ctx context.Context,
	m *models.Message,
	sender SmbSender,
	r models.Recipient,
) error {
	var sendErr error

	rcpt := r.RemotePath
	if rcpt == nil {
		return errEmptyRecipient
	}

	if *rcpt == "" {
		return errEmptyRecipient
	}

	for _, f := range append(m.Files, m.Images...) {
		err := sender.Upload(ctx, *rcpt, f)
		if err != nil {
			sendErr = errors.Join(sendErr, err)
		}
	}

	return sendErr
}

func sendSMTP(
	ctx context.Context,
	m *models.Message,
	sender SMTPSender,
	r models.Recipient,
) error {
	rcpt := r.Email
	if rcpt == nil {
		return errEmptyRecipient
	}

	subj, err := text.ExecuteTemplate(rcpt.Subject, m.Meta)
	if err != nil {
		subj = rcpt.Subject
	}

	body, err := text.ExecuteTemplate(deRef(rcpt.Body, ""), m.Meta)
	if err != nil {
		slog.Default().Error("email body template", slog.Any("error", err))

		if len(m.Text) > 0 {
			body = m.Text[0].Data.String()
		} else {
			body = deRef(rcpt.Body, "")
		}
	}

	mail := smtp.Mail{
		Recipients:  rcpt.Dest,
		Copy:        rcpt.Copy,
		Subject:     subj,
		Body:        body,
		Attachments: []smtp.Attachment{},
	}

	for _, f := range append(m.Files, m.Images...) {
		mail.Attachments = append(mail.Attachments, smtp.Attachment{
			File: f.Data,
			Name: f.FileName,
		})
	}

	return sender.Send(ctx, mail)
}

func deRef[T any](t *T, els T) T {
	if t == nil {
		return els
	}

	return *t
}

package models

import (
	"context"
	"errors"
	"fmt"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/pkg/text"
)

var ErrEmptyRecipient = errors.New("empty recipient")

type tgSender interface {
	SendText(ctx context.Context, rcpt TgChat, s string) (*TgMessage, error)
	SendDocument(ctx context.Context, rcpt TgChat, model []Data) ([]TgMessage, error)
	SendMedia(ctx context.Context, rcpt TgChat, model []Data) ([]TgMessage, error)
}

type smbSender interface {
	Upload(ctx context.Context, remote string, data Data) error
}

type smtpSender interface {
	Send(ctx context.Context, mail smtp.Mail) error
}

type senderProvider interface {
	Tg() tgSender
	SMB() smbSender
	SMTP() smtpSender
}

type Message struct {
	ReportName string

	Recipients []Recipient

	Text   []Data
	Files  []Data
	Images []Data
}

func NewMessage(rName string, data []Data, rcpts ...Recipient) *Message {
	var txt, fl, imgs []Data
	for _, d := range data {
		switch d.Type {
		case SendTextKind:
			txt = append(txt, d)
		case SendFileKind:
			fl = append(fl, d)
		case SendImageKind:
			imgs = append(imgs, d)
		default:
		}
	}

	return &Message{
		ReportName: rName,
		Recipients: rcpts,
		Text:       txt,
		Files:      fl,
		Images:     imgs,
	}
}

func (m *Message) Send(ctx context.Context, sp senderProvider) ([]TgMessage, error) {
	var sendErr error
	var tgMsg []TgMessage
	for _, r := range m.Recipients {
		switch r.Type {
		case TelegramRecipient:
			msg, err := m.SendTg(ctx, sp.Tg(), r)
			if err != nil {
				sendErr = errors.Join(sendErr, err)
				continue
			}
			if r.NeedDeleteAfterEndOfDay {
				tgMsg = append(tgMsg, msg...)
			}
		case EmailRecipient:
			err := m.SendSMTP(ctx, sp.SMTP(), r)
			if err != nil {
				sendErr = errors.Join(sendErr, err)
				continue
			}
		case SambaRecipient:
			err := m.SendSMB(ctx, sp.SMB(), r)
			if err != nil {
				sendErr = errors.Join(sendErr, err)
				continue
			}

		default:
			sendErr = errors.Join(sendErr, fmt.Errorf("unsupported recipient type: %s", r.Type))

		}
	}

	return tgMsg, sendErr
}

func (m *Message) SendTg(ctx context.Context, sender tgSender, r Recipient) ([]TgMessage, error) {
	if r.Chat == nil {
		return nil, ErrEmptyRecipient
	}

	rcpt := TgChat{ChatID: r.Chat.ChatID, ThreadID: deRef(r.ThreadID, 0)}

	var sendErr error

	var retMsg []TgMessage

	if m.Text != nil {
		for _, data := range m.Text {
			msg, err := sender.SendText(ctx, rcpt, data.Data.String())
			if err != nil {
				sendErr = fmt.Errorf("sending text: %w", err)
			} else {
				retMsg = append(retMsg, TgMessage{
					MessageID: msg.MessageID,
					Time:      msg.Time,
					ChatID:    msg.ChatID,
					ThreadID:  msg.ThreadID,
				})
			}
		}
	}

	if m.Files != nil {
		msg, err := sender.SendDocument(ctx, rcpt, m.Files)
		if err != nil {
			sendErr = fmt.Errorf("sending document: %w", err)
		} else {
			for _, m := range msg {
				retMsg = append(retMsg, TgMessage{
					MessageID: m.MessageID,
					Time:      m.Time,
					ChatID:    m.ChatID,
					ThreadID:  m.ThreadID,
				})
			}
		}
	}

	if m.Images != nil {
		msg, err := sender.SendMedia(ctx, rcpt, m.Images)
		if err != nil {
			sendErr = fmt.Errorf("sending document: %w", err)
		} else {
			for _, m := range msg {
				retMsg = append(retMsg, TgMessage{
					MessageID: m.MessageID,
					Time:      m.Time,
					ChatID:    m.ChatID,
					ThreadID:  m.ThreadID,
				})
			}
		}
	}

	return retMsg, sendErr
}

func (m *Message) SendSMB(ctx context.Context, sender smbSender, r Recipient) error {
	var sendErr error

	rcpt := r.RemotePath
	if rcpt == nil {
		return ErrEmptyRecipient
	}

	if *rcpt == "" {
		return ErrEmptyRecipient
	}

	for _, f := range append(m.Files, m.Images...) {
		err := sender.Upload(ctx, *rcpt, f)
		if err != nil {
			sendErr = errors.Join(sendErr, err)
		}
	}

	return sendErr
}

func (m *Message) SendSMTP(ctx context.Context, sender smtpSender, r Recipient) error {
	rcpt := r.Email
	if rcpt == nil {
		return ErrEmptyRecipient
	}

	subj, err := text.ExecuteTemplate(rcpt.Subject, nil)
	if err != nil {
		subj = rcpt.Subject
	}

	body, err := text.ExecuteTemplate(deRef(rcpt.Body, ""), nil)
	if err != nil {
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
			Name: f.Name,
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

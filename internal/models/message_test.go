package models

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"support_bot/internal/delivery/smtp"
)

type fakeTg struct{ TgSender }

func (fakeTg) SendText(context.Context, TgChat, string) (*SentMessage, error) {
	return &SentMessage{MessageID: 1}, nil
}

type fakeSMTP struct{ err error }

func (f fakeSMTP) Send(context.Context, smtp.Mail) error { return f.err }

type fakeProvider struct{ smtpErr error }

func (fakeProvider) Tg() TgSender       { return fakeTg{} }
func (fakeProvider) SMB() SmbSender     { return nil }
func (f fakeProvider) SMTP() SMTPSender { return fakeSMTP{err: f.smtpErr} }
func (fakeProvider) MAX() MaxSender     { return nil }

func TestMessageSendDelivered(t *testing.T) {
	tg := Recipient{
		Type:                    TelegramRecipient,
		Chat:                    &Chat{ChatID: 1, ChType: ChatTypeTg},
		NeedDeleteAfterEndOfDay: true,
	}
	mail := Recipient{
		Type:  EmailRecipient,
		Email: &EmailTemplate{Dest: []string{"a@b.c"}},
	}
	text := []Data{{Data: bytes.NewBufferString("hi"), Type: SendTextKind}}

	tests := []struct {
		name      string
		rcpts     []Recipient
		smtpErr   error
		delivered int
		wantErr   bool
		wantSent  int
	}{
		{
			"tg ok, email fails",
			[]Recipient{tg, mail},
			errors.New("boom"),
			1,
			true,
			1,
		},
		{"both ok", []Recipient{tg, mail}, nil, 2, false, 1},
		{"unsupported type", []Recipient{{Type: "bogus"}}, nil, 0, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMessage("r", text, nil, tt.rcpts...)

			sent, delivered, err := m.Send(
				context.Background(),
				fakeProvider{smtpErr: tt.smtpErr},
			)
			if delivered != tt.delivered || (err != nil) != tt.wantErr ||
				len(sent) != tt.wantSent {
				t.Fatalf(
					"delivered=%d err=%v sent=%d",
					delivered,
					err,
					len(sent),
				)
			}
		})
	}
}

type fakeMaxSender struct {
	MaxSender

	msg *SentMessage
	err error
}

func (f fakeMaxSender) SendDocument(
	context.Context,
	MaxChat,
	[]Data,
) (*SentMessage, error) {
	return f.msg, f.err
}

type maxProvider struct {
	fakeProvider

	max MaxSender
}

func (p maxProvider) MAX() MaxSender { return p.max }

func TestMessageSendMaxPartial(t *testing.T) {
	rcpt := Recipient{
		Type:                    MaxRecipient,
		Chat:                    &Chat{ChatID: 1, ChType: ChatTypeMax},
		NeedDeleteAfterEndOfDay: true,
	}
	files := []Data{{
		Data:     bytes.NewBufferString("x"),
		FileName: "a.txt",
		Type:     sendFileKind,
	}}
	m := NewMessage("r", files, nil, rcpt)

	sent, delivered, err := m.Send(
		context.Background(),
		maxProvider{max: fakeMaxSender{
			msg: &SentMessage{MessageID: 7},
			err: errors.New("upload a.txt"),
		}},
	)
	if err == nil || delivered != 0 || len(sent) != 1 {
		t.Fatalf("delivered=%d err=%v sent=%d", delivered, err, len(sent))
	}
}

package delivery

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"support_bot/internal/delivery/smtp"
	"support_bot/internal/models"
)

type fakeTg struct{ TgSender }

func (fakeTg) SendText(
	context.Context,
	models.TgChat,
	string,
) (*models.SentMessage, error) {
	return &models.SentMessage{MessageID: 1}, nil
}

type fakeSMTP struct{ err error }

func (f fakeSMTP) Send(context.Context, smtp.Mail) error { return f.err }

func TestDispatcherSendDelivered(t *testing.T) {
	tg := models.Recipient{
		Type: models.TelegramRecipient,
		Chat: &models.Chat{
			ChatID: 1,
			ChType: models.ChatTypeTg,
		},
		NeedDeleteAfterEndOfDay: true,
	}
	mail := models.Recipient{
		Type:  models.EmailRecipient,
		Email: &models.EmailTemplate{Dest: []string{"a@b.c"}},
	}
	text := []models.Data{
		{Data: bytes.NewBufferString("hi"), Type: models.SendTextKind},
	}

	tests := []struct {
		name      string
		rcpts     []models.Recipient
		smtpErr   error
		delivered int
		wantErr   bool
		wantSent  int
	}{
		{
			"tg ok, email fails",
			[]models.Recipient{tg, mail},
			errors.New("boom"),
			1,
			true,
			1,
		},
		{"both ok", []models.Recipient{tg, mail}, nil, 2, false, 1},
		{
			"unsupported type",
			[]models.Recipient{{Type: "bogus"}},
			nil,
			0,
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := models.NewMessage("r", text, nil, tt.rcpts...)
			d := NewDispatcher(fakeTg{}, nil, fakeSMTP{err: tt.smtpErr}, nil)

			sent, delivered, err := d.Send(context.Background(), m)
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

	msg *models.SentMessage
	err error
}

func (f fakeMaxSender) SendDocument(
	context.Context,
	models.MaxChat,
	[]models.Data,
) (*models.SentMessage, error) {
	return f.msg, f.err
}

func TestDispatcherSendMaxPartial(t *testing.T) {
	rcpt := models.Recipient{
		Type: models.MaxRecipient,
		Chat: &models.Chat{
			ChatID: 1,
			ChType: models.ChatTypeMax,
		},
		NeedDeleteAfterEndOfDay: true,
	}

	file, err := models.NewFileData(bytes.NewBufferString("x"), "a.txt")
	if err != nil {
		t.Fatal(err)
	}

	m := models.NewMessage("r", []models.Data{file}, nil, rcpt)
	d := NewDispatcher(nil, nil, nil, fakeMaxSender{
		msg: &models.SentMessage{MessageID: 7},
		err: errors.New("upload a.txt"),
	})

	sent, delivered, err := d.Send(context.Background(), m)
	if err == nil || delivered != 0 || len(sent) != 1 {
		t.Fatalf("delivered=%d err=%v sent=%d", delivered, err, len(sent))
	}
}

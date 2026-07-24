package models

import (
	"time"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/mymmrac/telego"
)

type TgChat struct {
	ChatID   int64
	ThreadID int
}

type SentMessage struct {
	ID int64 `db:"id"`

	MessageID    int     `db:"message_id"`
	MessageIDStr *string `db:"message_id_str"`

	Time time.Time `db:"sent_at"`

	ChatID int64 `db:"chat_id"`

	ThreadID int `db:"thread_id"`

	Title string `db:"title"`

	Deleted bool   `db:"deleted"`
	ChType  string `db:"ch_type"`
}

func NewFromTelego(msg *telego.Message) *SentMessage {
	if msg == nil {
		return nil
	}

	return &SentMessage{
		MessageID: msg.MessageID,
		Time:      time.Unix(msg.GetDate(), 0),
		ChatID:    msg.Chat.ID,
		ThreadID:  msg.MessageThreadID,
		Title:     msg.Chat.Title,
		ChType:    ChatTypeTg,
	}
}

func NewFromTelegoMany(msgs []telego.Message) []SentMessage {
	retMsg := make([]SentMessage, 0, len(msgs))
	for _, msg := range msgs {
		retMsg = append(retMsg, *NewFromTelego(&msg))
	}

	return retMsg
}

func NewMsgFromMax(max model.Message) *SentMessage {
	return &SentMessage{
		MessageIDStr: &max.Body.Mid,
		Time:         time.UnixMilli(max.Timestamp),
		ChatID:       max.Recipient.ChatID,
		ThreadID:     0,
		Title:        "max chat",
		ChType:       ChatTypeMax,
	}
}

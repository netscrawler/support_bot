package models

import (
	"time"

	"gopkg.in/telebot.v4"
)

type TgChat struct {
	ChatID   int64
	ThreadID int
}

type TgMessage struct {
	ID        int64 `db:"id"`
	MessageID int   `db:"message_id"`

	Time time.Time `db:"sent_at"`

	ChatID int64 `db:"chat_id"`

	ThreadID int `db:"thread_id"`

	Title string `db:"title"`

	Deleted bool `db:"deleted"`
}

func NewFromTelebot(msg *telebot.Message) *TgMessage {
	if msg == nil {
		return nil
	}

	return &TgMessage{
		MessageID: msg.ID,
		Time:      msg.Time(),
		ChatID:    msg.Chat.ID,
		ThreadID:  msg.ThreadID,
		Title:     msg.Chat.Title,
	}
}

func NewMsgFromTelebotMany(msgs []telebot.Message) []TgMessage {
	retMsg := make([]TgMessage, 0, len(msgs))
	for _, msg := range msgs {
		retMsg = append(retMsg, *NewFromTelebot(&msg))
	}

	return retMsg
}

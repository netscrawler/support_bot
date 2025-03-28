package models

import "gopkg.in/telebot.v4"

// Chat представляет чат для отправки уведомлений
type Chat struct {
	ID          int    `json:"id"`
	ChatID      int64  `json:"chat_id"`
	Title       string `json:"title"`
	Type        string `json:"type"` // 'private', 'group', 'supergroup', 'channel'
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

func NewChat(chat *telebot.Chat) *Chat {
	return &Chat{
		ChatID:      chat.ID,
		Title:       chat.Title,
		Type:        string(chat.Type),
		Description: chat.Description,
	}
}

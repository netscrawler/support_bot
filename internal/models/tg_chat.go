package models

import "fmt"

// TgChatDTO представляет чат для отправки уведомлений.
type TgChatDTO struct {
	ID          int     `db:"id"          json:"id"`
	ChatID      int64   `db:"chat_id"     json:"chat_id"`
	Title       string  `db:"title"       json:"title"`
	Type        string  `db:"type"        json:"type"` // 'private', 'group', 'supergroup', 'channel'
	Description *string `db:"description" json:"description"`
	IsActive    bool    `db:"is_active"   json:"is_active"`
	ThreadID    int64
}

func NewTgChatDTO(id int64, title, cType, desc string) *TgChatDTO {
	return &TgChatDTO{
		ChatID:      id,
		Title:       title,
		Type:        cType,
		Description: &desc,
	}
}

func (c *TgChatDTO) Activate() {
	c.IsActive = true
}

func (c *TgChatDTO) DeActivate() {
	c.IsActive = false
}

func (c *TgChatDTO) String() string {
	return fmt.Sprintf(`Чат: 
	ID: %d
	Title: %s`, c.ChatID, c.Title)
}

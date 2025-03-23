package models

import (
	"time"
)

// Chat представляет чат для отправки уведомлений
type Chat struct {
	ID            int       `json:"id"`
	ChatID        int64     `json:"chat_id"`
	Title         string    `json:"title"`
	Type          string    `json:"type"` // 'private', 'group', 'supergroup', 'channel'
	Description   string    `json:"description"`
	IsActive      bool      `json:"is_active"`
	AddedByUserID int       `json:"added_by_user_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

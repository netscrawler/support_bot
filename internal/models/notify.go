package models

import (
	"time"
)

// Notification представляет уведомление для отправки
type Notification struct {
	ID              int       `json:"id"`
	Content         string    `json:"content"`
	CreatedByUserID int       `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

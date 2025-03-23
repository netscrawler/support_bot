package models

import (
	"time"
)

// ChatNotification представляет связь между уведомлением и чатом, куда оно отправлено
type ChatNotification struct {
	ID             int        `json:"id"`
	ChatID         int        `json:"chat_id"`
	NotificationID int        `json:"notification_id"`
	Status         string     `json:"status"` // 'pending', 'sent', 'failed'
	SentAt         *time.Time `json:"sent_at,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
}

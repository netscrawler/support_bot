package models

import (
	"time"
)

type User struct {
	ID            int       `json:"id"`
	TelegramID    int64     `json:"telegram_id"`
	Username      string    `json:"username"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Role          string    `json:"role"` // admin или user
	IsActive      bool      `json:"is_active"`
	AddedByUserID *int      `json:"added_by_user_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

package models

import (
	"math/rand/v2"

	"gopkg.in/telebot.v4"
)

const (
	AdminRole = "admin"
	UserRole  = "user"
)

const (
	Denied = "denied"
)

type User struct {
	ID         int     `json:"id"`
	TelegramID int64   `json:"telegram_id"`
	Username   string  `json:"username"`
	FirstName  string  `json:"first_name"`
	LastName   *string `json:"last_name"`
	Role       string  `json:"role"` // admin или user
}

func NewUser(usr *telebot.User, isAdmin bool) *User {
	if isAdmin {
		return &User{
			TelegramID: usr.ID,
			Username:   usr.Username,
			FirstName:  usr.FirstName,
			LastName:   &usr.LastName,
			Role:       AdminRole,
		}
	}
	return &User{
		TelegramID: usr.ID,
		Username:   usr.Username,
		FirstName:  usr.FirstName,
		LastName:   &usr.LastName,
		Role:       UserRole,
	}
}

func NewEmptyUser(username string) *User {
	return &User{
		TelegramID: rand.Int64(),
		Username:   username,
		Role:       UserRole,
	}
}

func (u *User) IsAdmin() bool {
	return u.Role == AdminRole
}

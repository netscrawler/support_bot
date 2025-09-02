package models

import (
	"math/rand/v2"
)

const (
	PrimaryAdminRole = "primary"
	AdminRole        = "admin"
	UserRole         = "user"
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
	Role       string  `json:"role"` // admin или user или primary
}

func NewUser(tgID int64, username, firstname string, lastname *string, isAdmin bool) User {
	u := User{
		TelegramID: tgID,
		Username:   username,
		FirstName:  firstname,
		LastName:   lastname,
		Role:       UserRole,
	}
	if isAdmin {
		u.Role = AdminRole
	}
	return u
}

//nolint:gosec
func NewEmptyUser(username string, isAdmin bool) User {
	role := UserRole
	if isAdmin {
		role = AdminRole
	}

	return User{
		TelegramID: rand.Int64(),
		Username:   username,
		Role:       role,
	}
}

func (u *User) IsAdmin() bool {
	return u.Role == AdminRole || u.Role == PrimaryAdminRole
}

func (u *User) IsPrimaryAdmin() bool {
	return u.Role == PrimaryAdminRole
}

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
	ID         int     `db:"id"          json:"id"`
	TelegramID int64   `db:"telegram_id" json:"telegram_id"`
	Username   string  `db:"username"    json:"username"`
	FirstName  string  `db:"first_name"  json:"first_name"`
	LastName   *string `db:"last_name"   json:"last_name"`
	Role       string  `db:"role"        json:"role"` // admin или user или primary
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

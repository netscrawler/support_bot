package models

import (
	"fmt"
	"support_bot/internal/domain/errorz"

	"github.com/google/uuid"
)

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"

	UserRoleInvalid UserRole = "invalid"
)

type UserRole string

func NewRole(role string) (UserRole, error) {
	ur := UserRole(role)
	if ur == UserRoleAdmin || ur == UserRoleUser {
		return ur, nil
	}

	return UserRoleInvalid, fmt.Errorf("invalid role: %s", role)
}

func (u UserRole) String() string {
	return string(u)
}

func (u UserRole) IsValid() bool {
	return u == UserRoleAdmin || u == UserRoleUser
}

func (u UserRole) IsAdmin() bool {
	return u == UserRoleAdmin
}

type UserShort struct {
	UserRole
	ID       uuid.UUID
	Username string
}

func NewUserShort(id uuid.UUID, username string, role string) (UserShort, error) {
	uRole, err := NewRole(role)
	if err != nil {
		return UserShort{}, err
	}

	return UserShort{
		ID:       id,
		Username: username,
		UserRole: uRole,
	}, nil
}

type User struct {
	UserRole

	ID       uuid.UUID
	Username string
	Email    string
	Password string
}

func NewUser(id string, username string, email string, role string, password string) (User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return User{}, fmt.Errorf("%w : %w", errorz.ErrInternalServer, err)
	}

	uRole, err := NewRole(role)
	if err != nil {
		return User{}, fmt.Errorf("%w : %w", errorz.ErrInternalServer, err)
	}

	return User{
		ID:       uid,
		Username: username,
		Email:    email,
		UserRole: uRole,
		Password: password,
	}, nil
}

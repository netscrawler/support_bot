package service

import (
	"context"
	"errors"
	"support_bot/internal/models"
)

type UserProvider interface {
	Create(ctx context.Context, user *models.User) error

	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetAll(ctx context.Context) ([]models.User, error)
	GetByTgID(ctx context.Context, id int64) (*models.User, error)

	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, tgID int64) error
}

// User репозиторий для работы с данными пользователя.
type User struct {
	repo UserProvider
}

func NewUser(repo UserProvider) *User {
	return &User{
		repo: repo,
	}
}

func (u *User) GetAll(ctx context.Context) ([]models.User, error) {
	users, err := u.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (u *User) IsAllowed(ctx context.Context, id int64) (string, error) {
	user, err := u.repo.GetByTgID(ctx, id)
	if err != nil {
		return models.Denied, err
	}

	return user.Role, nil
}

func (u *User) GetAllUserIds(ctx context.Context) ([]int64, []int64, error) {
	users, err := u.repo.GetAll(ctx)
	if err != nil {
		return nil, nil, err
	}

	var userIds, adminIds []int64

	for _, user := range users {
		if user.IsAdmin() {
			adminIds = append(adminIds, user.TelegramID)

			continue
		}

		userIds = append(userIds, user.TelegramID)
	}

	return userIds, adminIds, nil
}

func (u *User) Create(ctx context.Context, user *models.User) error {
	const op = "service.User.Create"

	err := u.repo.Create(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) CreateEmpty(ctx context.Context, username string, isAdmin bool) error {
	const op = "service.User.Create"

	user := models.NewEmptyUser(username, isAdmin)

	err := u.repo.Create(ctx, &user)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) Update(usr *models.User) error {
	return u.repo.Update(context.Background(), usr)
}

func (u *User) AddUserComplete(user *models.User) error {
	return u.Update(user)
}

func (u *User) Delete(ctx context.Context, username string, primeReq bool) error {
	user, err := u.repo.GetByUsername(ctx, username)
	if errors.Is(err, models.ErrNotFound) {
		return err
	}

	if err != nil {
		return models.ErrInternal
	}

	if user.IsPrimaryAdmin() {
		return errors.New("not allowed")
	}

	if user.IsAdmin() && !primeReq {
		return errors.New("deleting admin is a sin")
	}

	err = u.repo.Delete(ctx, user.TelegramID)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) GetByUsername(ctx context.Context, uname string) (*models.User, error) {
	return u.repo.GetByUsername(ctx, uname)
}

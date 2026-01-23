package service

import (
	"context"
	"errors"
	"log/slog"

	models "support_bot/internal/models/notify"
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
	log  *slog.Logger
}

func NewUser(repo UserProvider, log *slog.Logger) *User {
	l := log.With(slog.Any("module", "tg_bot.service.user"))
	return &User{
		repo: repo,
		log:  l,
	}
}

func (u *User) GetAll(ctx context.Context) ([]models.User, error) {
	users, err := u.repo.GetAll(ctx)
	if err != nil {
		u.log.ErrorContext(ctx, "loading all users", slog.Any("error", err))
		return nil, err
	}

	return users, nil
}

func (u *User) IsAllowed(ctx context.Context, id int64) (string, error) {
	user, err := u.repo.GetByTgID(ctx, id)
	if err != nil {
		u.log.ErrorContext(ctx, "allowed check failed", slog.Any("error", err))
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
	err := u.repo.Create(ctx, user)
	if err != nil {
		u.log.ErrorContext(ctx, "create user", slog.Any("error", err))
		return err
	}

	return nil
}

func (u *User) CreateEmpty(ctx context.Context, username string, isAdmin bool) error {
	user := models.NewEmptyUser(username, isAdmin)

	err := u.repo.Create(ctx, &user)
	if err != nil {
		u.log.ErrorContext(ctx, "create empty user", slog.Any("error", err))
		return err
	}

	return nil
}

func (u *User) Update(ctx context.Context, usr *models.User) error {
	if err := u.repo.Update(ctx, usr); err != nil {
		u.log.ErrorContext(ctx, "updating user", slog.Any("error", err))
		return err
	}
	return nil
}

func (u *User) AddUserComplete(ctx context.Context, user *models.User) error {
	return u.Update(ctx, user)
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

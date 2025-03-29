package service

import (
	"context"
	"errors"
	"support_bot/internal/models"
	"support_bot/internal/repository"

	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

// User репозиторий для работы с данными пользователя
type User struct {
	repo *repository.User
	log  *zap.Logger
}

func NewUser(repo *repository.User, log *zap.Logger) *User {
	return &User{
		repo: repo,
		log:  log,
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
	user, err := u.repo.GetByTgId(ctx, id)
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
		if user.Role == models.AdminRole {
			adminIds = append(adminIds, user.TelegramID)
			continue
		}
		userIds = append(userIds, user.TelegramID)
	}
	return userIds, adminIds, nil
}

func (u *User) Create(ctx context.Context, tgId int64, username, firstName, lastName string) error {
	const op = "service.User.Create"

	user := &models.User{
		TelegramID: tgId,
		Username:   username,
		FirstName:  firstName,
		LastName:   &lastName,
		Role:       models.UserRole,
	}
	err := u.repo.Create(ctx, user)
	if err != nil {
		u.log.Info(op, zap.Error(err))
		return err
	}

	return nil
}

func (u *User) Update(usr *models.User) error {
	return u.repo.Update(context.Background(), usr)
}

func (s *User) AddUserComplete(user *telebot.User) error {
	usr := models.NewUser(user, false)

	return s.Update(usr)
}

func (u *User) Delete(ctx context.Context, username string) error {
	user, err := u.repo.GetByUsername(ctx, username)
	if errors.Is(err, models.ErrNotFound) {
		return err
	}
	if err != nil {
		return models.ErrInternal
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

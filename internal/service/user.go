package service

import (
	"context"
	"errors"
	"fmt"
	"support_bot/internal/models"
	"support_bot/internal/repository"

	"go.uber.org/zap"
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
	const op = "service.User.GetAll"
	users, err := u.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (u *User) CheckAccess(ctx context.Context, uid int64) string {
	const op = "service.User.CheckAccess"
	user, err := u.repo.GetByTgId(ctx, uid)
	if errors.Is(err, models.ErrNotFound) {
		return models.Denied
	}
	u.log.Info(op, zap.Any("user", *user), zap.Error(err))
	if err == nil {
		u.log.Info(fmt.Sprintf("%s acces granted for", op), zap.Any("user", user))
		return user.Role
	}

	return models.Denied
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

func (s *User) AddUserComplete(user models.User) error {
	// Check if user already exists

	// Update existing user with telegram_id if needed
	existingUser, err := s.repo.GetByUsername(context.TODO(), user.Username)
	if errors.Is(err, models.ErrNotFound) {
		return s.repo.Create(context.TODO(), &user)
	}
	if err != nil {
		return err
	}

	if existingUser.TelegramID == 0 {
		// Update the record with the telegram_id
		existingUser.TelegramID = user.TelegramID
		existingUser.FirstName = user.FirstName
		existingUser.LastName = user.LastName
		return s.repo.Update(context.TODO(), existingUser)

	}
	return s.repo.Create(context.Background(), &user)

	// Create new user
}

func (u *User) Delete(ctx context.Context, username string) error {
	const op = "service.User.Delete"

	user, err := u.repo.GetByUsername(ctx, username)
	if errors.Is(err, models.ErrNotFound) {
		return err
	}
	if err != nil {
		return models.ErrInternal
	}

	err = u.repo.Delete(ctx, user.TelegramID)
	if err != nil {
		u.log.Info(op, zap.Error(err))
		return err
	}
	return nil
}

func (u *User) GetByUsername(ctx context.Context, uname string) (*models.User, error) {
	return u.repo.GetByUsername(ctx, uname)
}

package user

import (
	"context"
	"support_bot/internal/domain/models"
)

type userProvider interface {
	GetByName(ctx context.Context, name string) (UserDBO, error)
	GetByID(ctx context.Context, name string) (UserDBO, error)
}

type UserService struct {
	repo userProvider
}

func NewUserService(repo userProvider) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByName(ctx context.Context, name string) (models.User, error) {
	if ctx.Err() != nil {
		return models.User{}, ctx.Err()
	}
	u, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return models.User{}, err
	}

	user, err := models.NewUser(u.ID, u.Login, u.Email, u.Role, u.Password)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, userID string) (models.User, error) {
	if ctx.Err() != nil {
		return models.User{}, ctx.Err()
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}

	user, err := models.NewUser(u.ID, u.Login, u.Email, u.Role, u.Password)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

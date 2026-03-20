package use_case

import (
	"context"
	"support_bot/internal/domain/errorz"
	domainModels "support_bot/internal/domain/models"
	domainSrv "support_bot/internal/domain/service"
)

type userProvider interface {
	GetByName(ctx context.Context, name string) (domainModels.User, error)
	GetByID(ctx context.Context, name string) (domainModels.User, error)
}

type tokenProvider interface {
	GenerateTokenPair(ctx context.Context, user domainModels.User) (string, string, error)
	CheckRefreshToken(ctx context.Context, token string) (string, error)
	CheckToken(ctx context.Context, token string) (domainModels.UserShort, error)
}

type Auth struct {
	userSrv  userProvider
	tokenSrv tokenProvider
}

func NewAuth(tokenSrv tokenProvider, userSrv userProvider) *Auth {
	return &Auth{tokenSrv: tokenSrv, userSrv: userSrv}
}

func (u *Auth) Login(ctx context.Context, login string, password string) (string, string, error) {
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}

	user, err := u.userSrv.GetByName(ctx, login)
	if err != nil {
		return "", "", err
	}

	ok, err := domainSrv.Verify(password, user.Password)
	if err != nil {
		return "", "", err
	}

	if !ok {
		return "", "", errorz.ErrInvalidPassword
	}

	token, refresh, err := u.tokenSrv.GenerateTokenPair(ctx, user)
	if err != nil {
		return "", "", err
	}

	return token, refresh, nil
}

func (u *Auth) Refresh(ctx context.Context, token string) (string, string, error) {
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}

	userID, err := u.tokenSrv.CheckRefreshToken(ctx, token)
	if err != nil {
		return "", "", err
	}

	user, err := u.userSrv.GetByID(ctx, userID)
	if err != nil {
		return "", "", err
	}

	accessToken, refreshToken, err := u.tokenSrv.GenerateTokenPair(ctx, user)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (u *Auth) CheckAdminAccess(ctx context.Context, token string) (string, bool, error) {
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}

	user, err := u.tokenSrv.CheckToken(ctx, token)
	if err != nil {
		return "", false, err
	}

	return user.ID.String(), user.IsAdmin() && user.IsValid(), nil
}

func (u *Auth) CheckUserAccess(ctx context.Context, token string) (string, bool, error) {
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}

	user, err := u.tokenSrv.CheckToken(ctx, token)
	if err != nil {
		return "", false, err
	}

	return user.ID.String(), user.IsValid(), nil
}

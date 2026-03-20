package tokens

import (
	"context"
	"fmt"
	"support_bot/internal/domain/errorz"
	"support_bot/internal/domain/models"
	"support_bot/internal/postgres"
)

type tokenProvider interface {
	postgres.UOW

	Save(ctx context.Context, token Token) error
	RevokeTokensByUserID(ctx context.Context, userID string) (int64, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (TokenDBO, error)
}

type TokenService struct {
	repo tokenProvider
	jwt  JWTService
}

func NewTokenService(repo tokenProvider, jwt JWTService) *TokenService {
	return &TokenService{repo, jwt}
}

func (ts *TokenService) GenerateTokenPair(ctx context.Context, user models.User) (string, string, error) {
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}

	uow, err := ts.repo.Start(ctx)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", errorz.ErrInternalServer, err)
	}

	_, err = ts.repo.RevokeTokensByUserID(uow, user.ID.String())
	if err != nil {
		return "", "", err
	}

	pair, err := ts.jwt.GenerateTokensPair(user)
	if err != nil {
		ts.repo.Rollback(uow)
		return "", "", err
	}

	token := NewToken(user.ID.String(), pair.Refresh, pair.Expires)

	err = ts.repo.Save(uow, token)
	if err != nil {
		return "", "", err
	}

	return pair.Access, pair.Refresh, ts.repo.Commit(uow)
}

func (ts *TokenService) CheckRefreshToken(ctx context.Context, token string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	err := ts.jwt.VerifyRefreshToken(token)
	if err != nil {
		return "", err
	}

	tokenHash := NewTokenHash(token)

	tokenDBO, err := ts.repo.FindByTokenHash(ctx, tokenHash.String())
	if err != nil {
		return "", err
	}

	if tokenDBO.Revoked {
		return "", errorz.ErrTokenExpired
	}

	return tokenDBO.UserID, nil
}

func (ts *TokenService) CheckToken(ctx context.Context, token string) (models.UserShort, error) {
	if ctx.Err() != nil {
		return models.UserShort{}, ctx.Err()
	}

	user, err := ts.jwt.VerifyAccessToken(token)
	if err != nil {
		return models.UserShort{}, err
	}

	return user, nil
}

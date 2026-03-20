package tokens_test

import (
	"testing"
	"time"

	"support_bot/internal/domain/errorz"
	"support_bot/internal/domain/models"
	"support_bot/internal/domain/tokens"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_VerifyToken(t *testing.T) {
	t.Parallel()

	t.Run("valid token", func(t *testing.T) {
		t.Parallel()
		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Minute,
		})

		inputUser := models.User{
			ID:       uuid.UUID{},
			Username: "username",
			Email:    "email@gmail.com",
			UserRole: "user", Password: "password",
		}

		token, err := srv.GenerateAccessToken(inputUser, uuid.NewString())
		require.NoError(t, err)

		user, err := srv.VerifyAccessToken(token)
		require.NoError(t, err)
		require.Equal(t, inputUser.Username, user.Username)
		require.Equal(t, inputUser.UserRole, user.UserRole)
		require.Equal(t, inputUser.ID, user.ID)
	})

	t.Run("malformed token", func(t *testing.T) {
		t.Parallel()

		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Minute,
		})

		_, err := srv.VerifyAccessToken("invalid token")
		require.Error(t, err)
		assert.ErrorIs(t, err, errorz.ErrInvalidToken)
	})

	t.Run("invalid claims", func(t *testing.T) {
		t.Parallel()

		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Minute,
		})

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": "username",
			"role":     "user",
		})

		signedToken, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		_, err = srv.VerifyAccessToken(signedToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, errorz.ErrInvalidToken)
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()

		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Millisecond,
		})

		inputUser := models.User{
			ID:       uuid.UUID{},
			Username: "username",
			Email:    "email@gmail.com",
			UserRole: "user",
			Password: "password",
		}

		token, err := srv.GenerateAccessToken(inputUser, uuid.NewString())
		require.NoError(t, err)

		// Wait a bit to ensure token is expired
		time.Sleep(10 * time.Millisecond)

		_, err = srv.VerifyAccessToken(token)
		require.Error(t, err)
		assert.ErrorIs(t, err, errorz.ErrTokenExpired)
	})

	t.Run("invalid issuer", func(t *testing.T) {
		t.Parallel()

		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Minute,
		})

		userID := uuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{
			Issuer:    "wrong_issuer",
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			ID:        uuid.NewString(),
		})

		signedToken, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		_, err = srv.VerifyAccessToken(signedToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, errorz.ErrInvalidToken)
	})

	t.Run("mismatched subject and user id", func(t *testing.T) {
		t.Parallel()

		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Minute,
		})

		type customClaims struct {
			jwt.RegisteredClaims
			UserID   uuid.UUID
			UserName string
			UserRole string
		}

		userID := uuid.New()
		differentID := uuid.New()

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &customClaims{
			UserID:   userID,
			UserName: "username",
			UserRole: "user",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "support_bot",
				Subject:   differentID.String(), // Different from UserID
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
				ID:        uuid.NewString(),
			},
		})

		signedToken, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		_, err = srv.VerifyAccessToken(signedToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, errorz.ErrInvalidToken)
	})

	t.Run("wrong secret", func(t *testing.T) {
		t.Parallel()

		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Minute,
		})

		inputUser := models.User{
			ID:       uuid.UUID{},
			Username: "username",
			Email:    "email@gmail.com",
			UserRole: "user",
			Password: "password",
		}

		type customClaims struct {
			jwt.RegisteredClaims
			UserID   uuid.UUID
			UserName string
			UserRole string
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &customClaims{
			UserID:   inputUser.ID,
			UserName: inputUser.Username,
			UserRole: inputUser.UserRole.String(),
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "support_bot",
				Subject:   inputUser.ID.String(),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
				ID:        uuid.NewString(),
			},
		})

		// Sign with different secret
		signedToken, err := token.SignedString([]byte("wrong_secret"))
		require.NoError(t, err)

		_, err = srv.VerifyAccessToken(signedToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, errorz.ErrInvalidToken)
	})

	t.Run("invalid user role", func(t *testing.T) {
		t.Parallel()

		srv := tokens.NewJWTService(tokens.JWTConfig{
			Secret:         "secret",
			AccessTokenTTL: time.Minute,
		})

		type customClaims struct {
			jwt.RegisteredClaims
			UserID   uuid.UUID
			UserName string
			UserRole string
		}

		userID := uuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &customClaims{
			UserID:   userID,
			UserName: "username",
			UserRole: "invalid_role", // Invalid role
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "support_bot",
				Subject:   userID.String(),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
				ID:        uuid.NewString(),
			},
		})

		signedToken, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		_, err = srv.VerifyAccessToken(signedToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, errorz.ErrInvalidToken)
	})
}

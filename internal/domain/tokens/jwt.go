package tokens

import (
	"errors"
	"fmt"
	"support_bot/internal/domain/errorz"
	"support_bot/internal/domain/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const issuer = "support_bot"

type claims struct {
	jwt.RegisteredClaims

	UserID   uuid.UUID
	UserName string
	UserRole string
}

type refreshClaims struct {
	jwt.RegisteredClaims
}

type JWTConfig struct {
	Secret string `yaml:"secret"`

	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`
}

type JWTService struct {
	JWTConfig
}

func NewJWTService(cfg JWTConfig) JWTService {
	return JWTService{cfg}
}

func (j *JWTService) GenerateTokensPair(user models.User) (tokenPair, error) {
	jti := uuid.NewString()

	accessToken, err := j.GenerateAccessToken(user, jti)
	if err != nil {
		return tokenPair{}, err
	}

	refreshToken, err := j.GenerateRefreshToken(user, jti)
	if err != nil {
		return tokenPair{}, err
	}

	return tokenPair{
		Access:  accessToken,
		Refresh: refreshToken,
		Expires: time.Now().Add(j.JWTConfig.RefreshTokenTTL),
	}, nil
}

func (j *JWTService) GenerateAccessToken(user models.User, jti string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims{
		UserID:   user.ID,
		UserRole: user.UserRole.String(),
		UserName: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.AccessTokenTTL)),
			ID:        jti,
		},
	})

	return token.SignedString([]byte(j.Secret))
}

func (j *JWTService) GenerateRefreshToken(user models.User, jti string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &refreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.RefreshTokenTTL)),
			ID:        jti,
		},
	})

	return token.SignedString([]byte(j.Secret))
}

func (j *JWTService) VerifyAccessToken(tokenString string) (models.UserShort, error) {
	var uClaims claims
	_, err := jwt.ParseWithClaims(tokenString, &uClaims, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.Secret), nil
	})
	if err != nil {
		// Check if the error is due to token expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			return models.UserShort{}, errorz.ErrTokenExpired
		}
		return models.UserShort{}, fmt.Errorf("%w : %w", errorz.ErrInvalidToken, err)
	}

	if uClaims.Subject != uClaims.UserID.String() {
		return models.UserShort{}, errorz.ErrInvalidToken
	}

	usr, err := models.NewUserShort(uClaims.UserID, uClaims.UserName, uClaims.UserRole)
	if err != nil {
		return models.UserShort{}, errorz.ErrInvalidToken
	}

	return usr, nil
}

func (j *JWTService) VerifyRefreshToken(tokenString string) error {
	var uClaims refreshClaims
	_, err := jwt.ParseWithClaims(tokenString, &uClaims, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.Secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return errorz.ErrTokenExpired
		}

		return fmt.Errorf("%w : %w", errorz.ErrInvalidToken, err)
	}

	return nil
}

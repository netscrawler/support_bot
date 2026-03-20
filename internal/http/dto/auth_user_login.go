package dto

import (
	"fmt"
	"support_bot/internal/http/errorz"

	val "github.com/go-playground/validator/v10"
)

type AuthUserRequestDTO struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (a AuthUserRequestDTO) Validate(v *val.Validate) error {
	err := v.Struct(a)
	if err != nil {
		return fmt.Errorf("%w : %w", errorz.ErrValidationError, err)
	}

	return nil
}

type AuthUserResponseDTO struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewAuthUserResponseDTOfromDomain(accessToken string, refreshToken string) AuthUserResponseDTO {
	return AuthUserResponseDTO{AccessToken: accessToken, RefreshToken: refreshToken}
}

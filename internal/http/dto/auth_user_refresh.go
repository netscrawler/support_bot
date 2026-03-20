package dto

import (
	"fmt"
	"support_bot/internal/http/errorz"

	val "github.com/go-playground/validator/v10"
)

type AuthUserRefreshRequestDTO struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (a AuthUserRefreshRequestDTO) Validate(v *val.Validate) error {
	err := v.Struct(a)
	if err != nil {
		return fmt.Errorf("%w : %w", errorz.ErrValidationError, err)
	}

	return nil
}

type AuthUserRefreshResponseDTO struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

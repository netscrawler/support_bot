package http

import (
	"support_bot/internal/domain/tokens"
	"support_bot/internal/http/middlewares"
)

type HTTPConfig struct {
	CORS middlewares.CORSConfig `yaml:"cors"`
	JWT  tokens.JWTConfig       `yaml:"jwt"`
}

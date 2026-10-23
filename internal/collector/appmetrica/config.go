package appmetrica

import (
	"support_bot/internal/pkg/secret"
	"time"
)

type Config struct {
	OAuthToken secret.String `yaml:"o_auth_token" env:"O_AUTH_TOKEN"`
	Timeout    time.Duration `yaml:"Timeout"      env:"TIMEOUT"`
}

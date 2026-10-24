package appmetrica

import (
	"time"

	"support_bot/internal/pkg/secret"
)

type Config struct {
	OAuthToken secret.String `env:"O_AUTH_TOKEN" json:"OAuthToken" yaml:"o_auth_token"`
	Timeout    time.Duration `env:"TIMEOUT"      json:"Timeout"    yaml:"Timeout"`
}

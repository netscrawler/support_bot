package appmetrica

import "time"

type Config struct {
	OAuthToken string        `yaml:"o_auth_token" env:"O_AUTH_TOKEN"`
	Timeout    time.Duration `yaml:"Timeout" env:"TIMEOUT"`
}

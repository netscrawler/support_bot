package jira

import (
	"support_bot/internal/pkg/secret"
	"time"
)

type Config struct {
	AuthToken secret.String `yaml:"auth_token" env:"AUTH_TOKEN"`
	JiraHost  string        `yaml:"jira_host"  env:"HOST"`
	Timeout   time.Duration `yaml:"timeout"    env:"TIMEOUT"`
}

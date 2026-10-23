package jira

import (
	"time"

	"support_bot/internal/pkg/secret"
)

type Config struct {
	AuthToken secret.String `env:"AUTH_TOKEN" json:"AuthToken" yaml:"auth_token"`
	JiraHost  string        `env:"HOST"       json:"JiraHost"  yaml:"jira_host"`
	Timeout   time.Duration `env:"TIMEOUT"    json:"Timeout"   yaml:"timeout"`
}

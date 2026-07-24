package jira

import "time"

type Config struct {
	AuthToken string        `yaml:"auth_token" env:"AUTH_TOKEN"`
	JiraHost  string        `yaml:"jira_host" env:"HOST"`
	Timeout   time.Duration `yaml:"timeout" env:"TIMEOUT"`
}

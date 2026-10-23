package zabbix

import (
	"support_bot/internal/pkg/secret"
	"time"
)

type Config struct {
	URL     string        `yaml:"url"     json:"url"     comment:"URL фронтенда Zabbix"`
	Token   secret.String `yaml:"token"   json:"token"   comment:"API-токен (роль с allow list *.get)"`
	Timeout time.Duration `yaml:"timeout" json:"timeout" comment:"Таймаут запроса (по умолчанию 30s)"`
}

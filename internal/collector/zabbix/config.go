package zabbix

import (
	"time"

	"support_bot/internal/pkg/secret"
)

type Config struct {
	URL     string        `comment:"URL фронтенда Zabbix"                                        json:"url"     yaml:"url"`
	Token   secret.String `comment:"API-токен (роль с allow list *.get)"                         json:"token"   yaml:"token"`
	Timeout time.Duration `comment:"Таймаут запроса (по умолчанию 30s)"                          json:"timeout" yaml:"timeout"`
}

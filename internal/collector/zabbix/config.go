package zabbix

import "time"

type Config struct {
	URL     string        `yaml:"url"     json:"url"     comment:"URL фронтенда Zabbix"`
	Token   string        `yaml:"token"   json:"token"   comment:"API-токен (роль с allow list *.get)"`
	Timeout time.Duration `yaml:"timeout" json:"timeout" comment:"Таймаут запроса (по умолчанию 30s)"`
}

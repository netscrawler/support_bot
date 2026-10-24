package httpcoll

import (
	"time"

	"support_bot/internal/pkg/secret"
)

type Config struct {
	BaseURL    string            `comment:"Базовый URL источника (http/https)"                                                               json:"base_url"    yaml:"base_url"`
	Auth       string            `comment:"Авторизация: bearer, basic или header (пусто — без неё)"                                          json:"auth"        yaml:"auth"`
	Token      secret.String     `comment:"Токен для bearer и header"                                                                        json:"token"       yaml:"token"`
	User       string            `comment:"Пользователь для basic"                                                                           json:"user"        yaml:"user"`
	Password   secret.String     `comment:"Пароль для basic"                                                                                 json:"password"    yaml:"password"`
	AuthHeader string            `comment:"Имя заголовка для auth: header"                                                                   json:"auth_header" yaml:"auth_header"`
	Headers    map[string]string `comment:"Дополнительные заголовки (токены — только через token)"                                           json:"headers"     yaml:"headers"`
	Timeout    time.Duration     `comment:"Таймаут запроса (по умолчанию 30s)"                                                               json:"timeout"     yaml:"timeout"`
}

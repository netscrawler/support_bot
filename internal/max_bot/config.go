package maxbot

import (
	"time"

	"support_bot/internal/pkg/secret"
)

type Config struct {
	Token       secret.String `comment:"Телеграмм токен бота полученный на платформе."                                                                                                                                             env:"MAX_TOKEN"            yaml:"max_token"`
	CleanUpTime time.Duration `comment:"CleanUpTime — интервал очистки временных данных бота\n(кэш, состояния диалогов, временные сообщения и т.п.)."                                                                              env:"MAX_CLEAN_UP_TIME"    env-default:"10m"  yaml:"clean_up_time"`
	BotPoll     time.Duration `comment:"BotPoll — интервал long-polling запросов к Max API."                                                                                                                                       env:"MAX_BOT_POLL_TIMEOUT" env-default:"30s"  yaml:"bot_poll"`
	APIProxy    string        `end:"API_PROXY"                                                                                                                                                                                     yaml:"api_proxy"`
	Enabled     bool          `comment:"Флаг активации бота, false по умолчанию. Отвечает за то включать бота или нет"                                                                                                             env:"MAX_ENABLED"          yaml:"max_enabled"`
}

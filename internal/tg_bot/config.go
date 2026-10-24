package tgbot

import (
	"time"

	"support_bot/internal/pkg/secret"
)

type Config struct {
	TelegramToken secret.String `comment:"Телеграмм токен бота полученый от @BotFather\nОбязателен для запуска бота."                                                                                                                env:"TELEGRAM_TOKEN"            yaml:"telegram_token"`
	CleanUpTime   time.Duration `comment:"CleanUpTime — интервал очистки временных данных бота\n(кэш, состояния диалогов, временные сообщения и т.п.)."                                                                              env:"TELEGRAM_CLEAN_UP_TIME"    env-default:"10m"     yaml:"clean_up_time"`
	BotPoll       time.Duration `comment:"BotPoll — интервал long-polling запросов к Telegram API."                                                                                                                                  env:"TELEGRAM_BOT_POLL_TIMEOUT" env-default:"30s"     yaml:"bot_poll"`
	Proxy         string        `env:"PROXY"                                                                                                                                                                                         yaml:"proxy"`
	APIProxy      string        `end:"API_PROXY"                                                                                                                                                                                     yaml:"api_proxy"`
}

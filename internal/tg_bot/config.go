package tgbot

import "time"

type Config struct {
	TelegramToken string        `env:"TELEGRAM_TOKEN"            yaml:"telegram_token" comment:"Телеграмм токен бота полученый от @BotFather\nОбязателен для запуска бота."`
	CleanUpTime   time.Duration `env:"TELEGRAM_CLEAN_UP_TIME"    yaml:"clean_up_time"  comment:"CleanUpTime — интервал очистки временных данных бота\n(кэш, состояния диалогов, временные сообщения и т.п.)." env-default:"10m"`
	BotPoll       time.Duration `env:"TELEGRAM_BOT_POLL_TIMEOUT" yaml:"bot_poll"       comment:"BotPoll — интервал long-polling запросов к Telegram API."                                                     env-default:"30s"`
	Proxy         string        `env:"PROXY"                     yaml:"proxy"`
	ApiProxy      string        `                                yaml:"api_proxy"                                                                                                                                               end:"API_PROXY"`
}

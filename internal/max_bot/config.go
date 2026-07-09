package maxbot

import "time"

type Config struct {
	Token       string        `env:"MAX_TOKEN"            yaml:"max_token"     comment:"Телеграмм токен бота полученный на платформе."`
	CleanUpTime time.Duration `env:"MAX_CLEAN_UP_TIME"    yaml:"clean_up_time" comment:"CleanUpTime — интервал очистки временных данных бота\n(кэш, состояния диалогов, временные сообщения и т.п.)." env-default:"10m"`
	BotPoll     time.Duration `env:"MAX_BOT_POLL_TIMEOUT" yaml:"bot_poll"      comment:"BotPoll — интервал long-polling запросов к Max API."                                                          env-default:"30s"`
	ApiProxy    string        `                                yaml:"api_proxy"                                                                                                                                               end:"API_PROXY"`
	Enabled     bool          `env:"MAX_ENABLED" yaml:"max_enabled" comment:"Флаг активации бота, false по умолчанию. Отвечает за то включать бота или нет"`
}

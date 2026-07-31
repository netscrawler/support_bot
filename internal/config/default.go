package config

import (
	"support_bot/internal/processor/lua"
	"time"

	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	maxbot "support_bot/internal/max_bot"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"
	tgbot "support_bot/internal/tg_bot"
)

func Default() *Config {
	return &Config{
		Log: logger.LogConfig{
			Level:  "prod",
			File:   "./log.log",
			Output: []string{"stdout"},
			Format: "text",
		},
		MetabaseDomain: "https://metabase.domain",
		AppMetrica: appmetrica.Config{
			OAuthToken: "auth token for appmetrica",
			Timeout:    5 * time.Minute,
		},
		Jira: jira.Config{
			AuthToken: "auth token from jira",
			JiraHost:  "https://your-jira-instance.atlassian.net",
			Timeout:   5 * time.Minute,
		},
		Lua: lua.Config{
			ExecutionTimeout: 5 * time.Minute,
			MaxMemoryMB:      256,
			AllowedModules: []string{
				"json",    // JSON кодирование/декодирование
				"http",    // HTTP запросы
				"url",     // парсинг URL
				"time",    // работа со временем
				"strings", // строковые операции
				"inspect", // отладка
			},
		},
		Database: postgres.Config{
			Port:            5432,
			Host:            "localhost",
			User:            "postgres",
			Password:        "postgres",
			Name:            "database_name",
			SSL:             "disable",
			MaxConns:        10,
			MaxIdleConns:    5,
			MaxConnLifeTime: 30 * time.Minute,
			MaxConnIdleTime: 2 * time.Minute,
			DatabaseConnect: 30 * time.Second,
		},
		TgBot: tgbot.Config{
			TelegramToken: "telegram_bot_token",
			CleanUpTime:   10 * time.Minute,
			BotPoll:       30 * time.Second,
		},
		MaxBot: maxbot.Config{
			Token:       "max_bot_token",
			CleanUpTime: 0,
			BotPoll:     0,
			ApiProxy:    "",
			Enabled:     false,
		},
		Timeout: timeout{
			Shutdown: 5 * time.Second,
		},
		SMB: smb.Config{
			Address:  "localhost:542",
			User:     "user",
			Password: "password",
			Domain:   "WORKGROUP",
			Share:    "public",
			Active:   true,
		},
		SMTP: smtp.Config{
			Host:     "smtp.example.com",
			Port:     "465",
			Email:    "example@example.com",
			Password: "password",
		},
	}
}

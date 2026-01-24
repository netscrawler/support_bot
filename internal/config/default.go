package config

import (
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"
	"time"
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
		Database: postgres.PostgresConfig{
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
		Bot: bot{
			TelegramToken: "telegram_bot_token",
			CleanUpTime:   10 * time.Minute,
			BotPoll:       30 * time.Second,
		},
		Timeout: timeout{
			Shutdown: 5 * time.Second,
		},
		SMB: smb.SMBConfig{
			Address:  "localhost:542",
			User:     "user",
			Password: "password",
			Domain:   "WORKGROUP",
			Share:    "public",
			Active:   true,
		},
		SMTP: smtp.SMTPConfig{
			Host:     "smtp.example.com",
			Port:     "465",
			Email:    "example@example.com",
			Password: "password",
		},
	}
}

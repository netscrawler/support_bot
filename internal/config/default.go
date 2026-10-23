package config

import (
	"time"

	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	pgcollector "support_bot/internal/collector/postgres"
	"support_bot/internal/collector/prometheus"
	"support_bot/internal/collector/zabbix"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/pkg/retry"
	"support_bot/internal/postgres"
	"support_bot/internal/processor/lua"
	"support_bot/internal/runlog"

	apihttp "support_bot/internal/api/http"

	maxbot "support_bot/internal/max_bot"

	tgbot "support_bot/internal/tg_bot"
)

func Default() *Config {
	return &Config{
		Log: logger.LogConfig{
			Level:   "prod",
			File:    "./log.log",
			Output:  []string{"stdout"},
			Format:  "text",
			Exclude: []string{"time"},
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
			Shutdown: 10 * time.Second,
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
		HTTP: apihttp.Config{
			Host:              "127.0.0.1",
			Port:              8080,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      6 * time.Minute,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
			MaxBodyBytes:      10 << 20,
		},
		Delivery: delivery{Retry: deliveryRetry{
			Delays: retry.Delays{time.Second, 5 * time.Second, 15 * time.Second},
		}},
		Runs: runlog.Config{Retention: 720 * time.Hour},
		Databases: map[string]pgcollector.Config{
			"example": {
				DSN:              "dsn://",
				ReadOnly:         new(true),
				StatementTimeout: 5 * time.Second,
				MaxConns:         5,
			},
			"example-2": {
				DSN:              "dsn://",
				ReadOnly:         new(true),
				StatementTimeout: 5 * time.Second,
				MaxConns:         5,
			},
		},
		Grafana: map[string]prometheus.GrafanaConfig{
			"example": {
				URL:     "https://grafana.example",
				Token:   "token",
				Timeout: 15 * time.Second,
			},
			"example-2": {
				URL:     "https://grafana.example",
				Token:   "token",
				Timeout: 15 * time.Second,
			},
		},
		Prometheus: map[string]prometheus.Config{
			"example": {
				URL:           "https://prometheus.org",
				BearerToken:   "token",
				Grafana:       "",
				DatasourceUID: "",
				Timeout:       0,
			},
			"example-2": {
				URL:         "https://prometheus.org",
				BearerToken: "",
				BasicAuth: &prometheus.BasicAuth{
					User:     "user",
					Password: "password",
				},
				Grafana:       "",
				DatasourceUID: "",
				Timeout:       15 * time.Second,
			},
		},
		Zabbix: map[string]zabbix.Config{
			"example": {
				URL:     "https://zabbix.org",
				Token:   "token",
				Timeout: 15 * time.Second,
			},
		},
		Alerts: runlog.AlertsConfig{
			ChatID:   13141412432,
			ThreadID: 12,
		},
	}
}

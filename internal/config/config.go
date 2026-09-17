package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"
	"support_bot/internal/processor/lua"
	"time"

	maxbot "support_bot/internal/max_bot"

	tgbot "support_bot/internal/tg_bot"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Log            logger.LogConfig  `yaml:"log"             comment:"Настройки логгирования"`
	MetabaseDomain string            `yaml:"metabase_domain" comment:"Адрес Metabase для забора данных"                                                                                                                                    env:"METABASE_DOMAIN"`
	AppMetrica     appmetrica.Config `yaml:"appmetrica"                                                                                                                                                                                    env:"APP_METRICA"`
	Jira           jira.Config       `yaml:"jira"                                                                                                                                                                                          env:"JIRA"`
	Lua            lua.Config        `yaml:"lua"             comment:"Настройки Lua-процессора."                                                                                                                                           env:"LUA"`
	Database       postgres.Config   `yaml:"database"        comment:"Настройки подключения к Postgres"`
	TgBot          tgbot.Config      `yaml:"telegram"        comment:"\nНастройки Telegram-бота.\nИспользуется для приема команд и отправки уведомлений."`
	Timeout        timeout           `yaml:"timeout"         comment:"Настройка таймаутов"`
	SMB            smb.Config        `yaml:"smb"             comment:"Настройки подключения к SMB (Samba) файловой шаре.\nИспользуется для чтения и/или записи файлов на сетевой ресурс.\nПоддерживается аутентификация по логину/паролю."`
	SMTP           smtp.Config       `yaml:"smtp"            comment:"Настройки SMTP-сервера.\nИспользуется для отправки email-уведомлений и отчетов.\nПоддерживается аутентификация по логину и паролю."`
	MaxBot         maxbot.Config     `yaml:"max"             comment:"Настройка Max бота"`
}

type timeout struct {
	Shutdown time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"5s" yaml:"shutdown" comment:"Shutdown — максимальное время на корректное завершение приложения.\nЗа это время должны завершиться все активные операции.\nЕсли указать слишком маленький период не все процеесы могут завершится корректно"`
}

// Load загружает конфигурацию из файла или из переменных окружения.
func Load(path string) (*Config, error) {
	var cfg Config

	configPath := fetchConfigPath(path)

	ext := filepath.Ext(configPath)

	if ext == ".env" {
		_ = godotenv.Load(configPath)
	} else {
		_ = godotenv.Load()
	}
	//nolint:errcheck //not need

	// Загрузка конфигурации
	if configPath != "" {
		// Если путь к файлу указан, загружаем из YAML
		err := cleanenv.ReadConfig(configPath, &cfg)
		if err != nil {
			return nil, fmt.Errorf("error readYaml config: %w", err)
		}
	} else {
		// Если путь не указан, загружаем из переменных окружения
		err := cleanenv.ReadEnv(&cfg)
		if err != nil {
			return nil, fmt.Errorf("error readEnv config: %w", err)
		}
	}

	return &cfg, nil
}

func (c Config) Validate() error {
	// TODO: add full config validation.
	return c.Log.Validate()
}

// Приоритет: 1) аргумент командной строки, 2) переменная окружения, 3) значение по умолчанию.
func fetchConfigPath(path string) string {
	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	if path == "" {
		path = "./config.yaml"
	}

	if path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return ""
		}
	}

	return path
}

type safeConfig Config

func (c Config) LogValue() slog.Value {
	c.Database.Password = strings.Repeat("*", len(c.Database.Password))
	c.TgBot.TelegramToken = strings.Repeat("*", len(c.TgBot.TelegramToken))
	c.MaxBot.Token = strings.Repeat("*", len(c.MaxBot.Token))
	c.Database.DSN = "postgres://***"
	c.SMB.Password = strings.Repeat("*", len(c.SMB.Password))
	c.SMTP.Password = strings.Repeat("*", len(c.SMTP.Password))
	c.Jira.AuthToken = strings.Repeat("*", len(c.Jira.AuthToken))
	c.AppMetrica.OAuthToken = strings.Repeat("*", len(c.AppMetrica.OAuthToken))

	return slog.AnyValue(safeConfig(c))
}

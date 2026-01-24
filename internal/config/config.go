package config

import (
	"fmt"
	"log/slog"
	"os"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Log            logger.LogConfig        `yaml:"log"             comment:"Настройки логгирования"`
	MetabaseDomain string                  `yaml:"metabase_domain" comment:"Адрес Metabase для забора данных"                                                                                                                                    env:"METABASE_DOMAIN"`
	Database       postgres.PostgresConfig `yaml:"database"        comment:"Настройки подключения к Postgres"`
	Bot            bot                     `yaml:"bot"             comment:"\nНастройки Telegram-бота.\nИспользуется для приема команд и отправки уведомлений."`
	Timeout        timeout                 `yaml:"timeout"         comment:"Настройка таймаутов"`
	SMB            smb.SMBConfig           `yaml:"smb"             comment:"Настройки подключения к SMB (Samba) файловой шаре.\nИспользуется для чтения и/или записи файлов на сетевой ресурс.\nПоддерживается аутентификация по логину/паролю."`
	SMTP           smtp.SMTPConfig         `yaml:"smtp"            comment:"Настройки SMTP-сервера.\nИспользуется для отправки email-уведомлений и отчетов.\nПоддерживается аутентификация по логину и паролю."`
}

type bot struct {
	TelegramToken string        `env:"TELEGRAM_TOKEN"            yaml:"telegram_token" comment:"Телеграмм токен бота полученый от @BotFather\nОбязателен для запуска бота."`
	CleanUpTime   time.Duration `env:"TELEGRAM_CLEAN_UP_TIME"    yaml:"clean_up_time"  comment:"CleanUpTime — интервал очистки временных данных бота\n(кэш, состояния диалогов, временные сообщения и т.п.)." env-default:"10m"`
	BotPoll       time.Duration `env:"TELEGRAM_BOT_POLL_TIMEOUT" yaml:"bot_poll"       comment:"BotPoll — интервал long-polling запросов к Telegram API."                                                     env-default:"30s"`
}

type timeout struct {
	Shutdown time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"5s" yaml:"shutdown" comment:"Shutdown — максимальное время на корректное завершение приложения.\nЗа это время должны завершиться все активные операции.\nЕсли указать слишком маленький период не все процеесы могут завершится корректно"`
}

// Load загружает конфигурацию из файла или из переменных окружения.
func Load() (*Config, error) {
	var cfg Config

	//nolint:errcheck //not need
	_ = godotenv.Load()

	configPath := fetchConfigPath()

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

var ConfigPath string

// Приоритет: 1) аргумент командной строки, 2) переменная окружения, 3) значение по умолчанию.
func fetchConfigPath() string {
	if ConfigPath == "" {
		ConfigPath = os.Getenv("CONFIG_PATH")
	}

	if ConfigPath == "" {
		ConfigPath = "./config.yaml"
	}

	if ConfigPath != "" {
		if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
			return ""
		}
	}

	return ConfigPath
}

type SafeConfig Config

func (c Config) LogValue() slog.Value {
	c.Database.Password = "***"
	c.Bot.TelegramToken = "***"
	c.Database.DSN = "postgres://***"
	c.SMB.Password = "***"
	c.SMTP.Password = "***"

	return slog.AnyValue(SafeConfig(c))
}

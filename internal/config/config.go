package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"
)

type Config struct {
	Log            logger.LogConfig        `yaml:"log"`
	MetabaseDomain string                  `yaml:"metabase_domain" env:"METABASE_DOMAIN"`
	Database       postgres.PostgresConfig `yaml:"database"`
	Bot            bot                     `yaml:"bot"`
	Timeout        timeout                 `yaml:"timeout"`
	SMB            smb.SMBConfig           `yaml:"smb"`
	SMTP           smtp.SMTPConfig         `yaml:"smtp"`
}

type bot struct {
	TelegramToken string        `env:"TELEGRAM_TOKEN"            yaml:"telegram_token"`
	CleanUpTime   time.Duration `env:"TELEGRAM_CLEAN_UP_TIME"    yaml:"clean_up_time"  env-default:"10m"`
	BotPoll       time.Duration `env:"TELEGRAM_BOT_POLL_TIMEOUT" yaml:"bot_poll"       env-default:"30s"`
}

type timeout struct {
	Shutdown time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"5s" yaml:"shutdown"`
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

// Приоритет: 1) аргумент командной строки, 2) переменная окружения, 3) значение по умолчанию.
func fetchConfigPath() string {
	var configPath string

	flag.StringVar(&configPath, "config", "", "Путь к файлу конфигурации")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	if configPath == "" {
		configPath = "./config.yaml"
	}

	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return ""
		}
	}

	return configPath
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

package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel       string   `yaml:"log_level"       env:"LOG_LEVEL"`
	MetabaseDomain string   `yaml:"metabase_domain" env:"METABASE_DOMAIN"`
	Database       database `yaml:"database"`
	Bot            bot      `yaml:"bot"`
	Timeout        timeout  `yaml:"timeout"`
	SMB            smb      `yaml:"smb"`
	SMTP           smtp     `yaml:"smtp"`
}

type smb struct {
	Adress string `yaml:"adress"   env:"ADRESS"`
	User   string `yaml:"user"     env:"USER"`
	PWD    string `yaml:"password" env:"PWD"`
	Domain string `yaml:"domain"   env:"DOMAIN"`
	Share  string `yaml:"share"    env:"SHARE"`
	Active bool   `yaml:"active"   env:"ACTIVE"`
}

type bot struct {
	TelegramToken string        `yaml:"telegram_token" env:"TELEGRAM_TOKEN"`
	CleanUpTime   time.Duration `yaml:"CleanUpTime"    env:"CLEAN_UP_TIME"  env-default:"10m"`
}

type database struct {
	Port     int    `yaml:"port"     env:"DATABASE_PORT"     env-default:"5432"`
	Host     string `yaml:"host"     env:"DATABASE_HOST"     env-default:"localhost"`
	User     string `yaml:"user"     env:"DATABASE_USER"     env-default:"user"`
	Password string `yaml:"password" env:"DATABASE_PASSWORD"`
	Name     string `yaml:"name"     env:"DATABASE_NAME"     env-default:"postgres"`
	URL      string
}

type timeout struct {
	DatabaseConnect time.Duration `yaml:"database_connect" env:"DATABASE_CONNECT_TIMEOUT" env-default:"30s"`
	BotPoll         time.Duration `yaml:"bot_poll"         env:"BOT_POLL_TIMEOUT"         env-default:"30s"`
	Shutdown        time.Duration `yaml:"shutdown"         env:"SHUTDOWN_TIMEOUT"         env-default:"5s"`
}

type smtp struct {
	Host     string `yaml:"smtp_serv"     env:"SMTP_SERV"`
	Port     string `yaml:"smtp_port"     env:"SMTP_PORT"`
	Email    string `yaml:"email"         env:"EMAIL"`
	Password string `yaml:"smtp_password" env:"SMTP_PASSWORD"`
}

// Load загружает конфигурацию из файла или из переменных окружения.
func Load() (*Config, error) {
	var cfg Config

	_ = godotenv.Load()
	configPath := fetchConfigPath()

	// Загрузка конфигурации
	if configPath != "" {
		// Если путь к файлу указан, загружаем из YAML
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			return nil, fmt.Errorf("error readYaml config: %w", err)
		}
	} else {
		// Если путь не указан, загружаем из переменных окружения
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("error readEnv config: %w", err)
		}
	}

	//nolint:nosprintfhostport
	cfg.Database.URL = fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.Name,
	)

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

package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Database configDatabase `yaml:"database"`
	Bot      configBot      `yaml:"bot"`
	App      configApp      `yaml:"app"`
	Timeout  configTimeout  `yaml:"timeout"`
}
type configBot struct {
	TelegramToken string `yaml:"telegram_token" env:"TELEGRAM_TOKEN"`
}
type configDatabase struct {
	Port     int    `yaml:"port"     env:"DATABASE_PORT"     env-default:"5432"`
	Host     string `yaml:"host"     env:"DATABASE_HOST"     env-default:"localhost"`
	User     string `yaml:"user"     env:"DATABASE_USER"     env-default:"user"`
	Password string `yaml:"password" env:"DATABASE_PASSWORD"`
	Name     string `yaml:"name"     env:"DATABASE_NAME"     env-default:"postgres"`
	URL      string
}

type configApp struct {
	Debug bool   `yaml:"debug" env:"APP_DEBUG" env-default:"false"`
	Host  string `yaml:"host"  env:"APP_HOST"  env-default:"localhost"`
	Port  string `yaml:"port"  env:"APP_PORT"  env-default:"8080"`
}

type configTimeout struct {
	DatabaseConnect time.Duration `yaml:"database_connect" env:"DATABASE_CONNECT_TIMEOUT" env-default:"30s"`
	BotPoll         time.Duration `yaml:"bot_poll"         env:"BOT_POLL_TIMEOUT"         env-default:"30s"`
	Shutdown        time.Duration `yaml:"shutdown"         env:"SHUTDOWN_TIMEOUT"         env-default:"5s"`
}

// Load загружает конфигурацию из файла или из переменных окружения
func Load() (*Config, error) {
	var cfg Config

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

	cfg.Database.URL = fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.Name,
	)

	cfg.Timeout.DatabaseConnect = time.Duration(cfg.Timeout.DatabaseConnect) * time.Second
	cfg.Timeout.BotPoll = time.Duration(cfg.Timeout.BotPoll) * time.Second
	cfg.Timeout.Shutdown = time.Duration(cfg.Timeout.Shutdown) * time.Second

	return &cfg, nil
}

// fetchConfigPath определяет путь к файлу конфигурации
// Приоритет: 1) аргумент командной строки, 2) переменная окружения, 3) значение по умолчанию
func fetchConfigPath() string {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Путь к файлу конфигурации")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return ""
		}
	}

	return configPath
}

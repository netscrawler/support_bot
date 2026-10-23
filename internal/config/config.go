package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	pgcollector "support_bot/internal/collector/postgres"
	"support_bot/internal/collector/prometheus"
	"support_bot/internal/collector/zabbix"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/metrics"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/pkg/retry"
	"support_bot/internal/pkg/secret"
	"support_bot/internal/postgres"
	"support_bot/internal/processor/lua"
	"support_bot/internal/runlog"
	"time"

	apihttp "support_bot/internal/api/http"

	maxbot "support_bot/internal/max_bot"

	tgbot "support_bot/internal/tg_bot"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Log            logger.LogConfig                    `yaml:"log"             comment:"Настройки логгирования"`
	MetabaseDomain string                              `yaml:"metabase_domain" comment:"Адрес Metabase для забора данных"                                                                                                                                                                  env:"METABASE_DOMAIN"`
	AppMetrica     appmetrica.Config                   `yaml:"appmetrica"                                                                                                                                                                                                                  env:"APP_METRICA"`
	Jira           jira.Config                         `yaml:"jira"                                                                                                                                                                                                                        env:"JIRA"`
	Lua            lua.Config                          `yaml:"lua"             comment:"Настройки Lua-процессора."                                                                                                                                                                         env:"LUA"`
	Database       postgres.Config                     `yaml:"database"        comment:"Настройки подключения к Postgres"`
	Databases      map[string]pgcollector.Config       `yaml:"databases"       comment:"Прямые подключения к Postgres для коллектора типа pg.\nКлюч — имя базы, которое указывается в card_uuid запроса."`
	Grafana        map[string]prometheus.GrafanaConfig `yaml:"grafana"         comment:"Подключения к Grafana: для запросов к Prometheus через proxy datasource и для коллектора grafana_alerts.\nКлюч — имя для поля grafana в секции prometheus и для card_uuid запроса grafana_alerts."`
	Prometheus     map[string]prometheus.Config        `yaml:"prometheus"      comment:"Источники Prometheus для коллектора типа prom.\nКлюч — имя источника, которое указывается в card_uuid запроса."`
	Zabbix         map[string]zabbix.Config            `yaml:"zabbix"          comment:"Источники Zabbix для коллектора типа zabbix.\nКлюч — имя источника, которое указывается в card_uuid запроса."`
	TgBot          tgbot.Config                        `yaml:"telegram"        comment:"Настройки Telegram-бота.\nИспользуется для приема команд и отправки уведомлений."`
	Timeout        timeout                             `yaml:"timeout"         comment:"Настройка таймаутов"`
	SMB            smb.Config                          `yaml:"smb"             comment:"Настройки подключения к SMB (Samba) файловой шаре.\nИспользуется для чтения и/или записи файлов на сетевой ресурс.\nПоддерживается аутентификация по логину/паролю."`
	SMTP           smtp.Config                         `yaml:"smtp"            comment:"Настройки SMTP-сервера.\nИспользуется для отправки email-уведомлений и отчетов.\nПоддерживается аутентификация по логину и паролю."`
	MaxBot         maxbot.Config                       `yaml:"max"             comment:"Настройка Max бота"`
	HTTP           apihttp.Config                      `yaml:"http"            comment:"Настройки публичного HTTP API."`
	Alerts         runlog.AlertsConfig                 `yaml:"alerts"          comment:"Уведомления администраторов о сбоях отчётов."`
	Runs           runlog.Config                       `yaml:"runs"            comment:"Журнал запусков отчётов."`
	Delivery       delivery                            `yaml:"delivery"        comment:"Доставка отчётов."`
	Metrics        metrics.Config                      `yaml:"metrics"         comment:"Сервер /metrics и /healthz."`
}

type delivery struct {
	Retry deliveryRetry `yaml:"retry"`
}

type deliveryRetry struct {
	Delays retry.Delays `yaml:"delays" env-default:"1s,5s,15s" comment:"Паузы между повторами неудавшейся отправки; попыток = пауз + 1. Пустой список — без повторов."`
}

type timeout struct {
	Shutdown time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"10s" yaml:"shutdown" comment:"Shutdown — максимальное время на корректное завершение приложения.\nЗа это время должны завершиться все активные операции.\nЕсли указать слишком маленький период не все процеесы могут завершится корректно"`
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

	registerSecrets(&cfg)

	return &cfg, nil
}

// registerSecrets передаёт все секреты конфига в secret.Redact.
func registerSecrets(cfg *Config) {
	secret.Register(
		cfg.Database.Password, cfg.TgBot.TelegramToken, cfg.MaxBot.Token,
		cfg.SMTP.Password, cfg.SMB.Password, cfg.Jira.AuthToken, cfg.AppMetrica.OAuthToken,
	)

	// Database.DSN здесь ещё пуст: его регистрирует postgres.Config.getDSN.
	for _, d := range cfg.Databases {
		registerDSN(d.DSN)
	}

	for _, p := range cfg.Prometheus {
		secret.Register(p.BearerToken)

		if p.BasicAuth != nil {
			secret.Register(p.BasicAuth.Password)
		}
	}

	for _, g := range cfg.Grafana {
		secret.Register(g.Token)
	}

	for _, z := range cfg.Zabbix {
		secret.Register(z.Token)
	}
}

// registerDSN регистрирует DSN целиком и пароль из него.
func registerDSN(dsn secret.String) {
	secret.Register(dsn)

	if u, err := url.Parse(dsn.Value()); err == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok {
			secret.Register(secret.String(pw))
		}
	}
}

// Validate проверяет обязательные поля: токен Telegram и БД всегда,
// остальные секции — только если задано их ключевое поле.
// Map-секции (databases, prometheus, grafana, zabbix) проверяют конструкторы коллекторов.
func (c Config) Validate() error {
	var errs []error

	need := func(path string, empty bool) {
		if empty {
			errs = append(errs, fmt.Errorf("%s: не задано", path))
		}
	}

	need("telegram.telegram_token", c.TgBot.TelegramToken == "")
	need("database.host", c.Database.Host == "")
	need("database.user", c.Database.User == "")
	need("database.name", c.Database.Name == "")

	if c.SMTP.Host != "" {
		need("smtp.email", c.SMTP.Email == "")
		need("smtp.password", c.SMTP.Password == "")
		need("smtp.port", c.SMTP.Port == "")
	}

	if c.SMB.Active {
		need("smb.address", c.SMB.Address == "")
		need("smb.user", c.SMB.User == "")
		need("smb.password", c.SMB.Password == "")
		need("smb.share", c.SMB.Share == "")
	}

	if c.MaxBot.Enabled {
		need("max.max_token", c.MaxBot.Token == "")
	}

	if c.Jira.JiraHost != "" {
		need("jira.auth_token", c.Jira.AuthToken == "")
	}

	return errors.Join(append(errs, c.Log.Validate(), c.HTTP.Validate())...)
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
		//nolint:gosec // путь конфига задаёт оператор
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return ""
		}
	}

	return path
}

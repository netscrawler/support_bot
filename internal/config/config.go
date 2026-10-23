package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

	return &cfg, nil
}

func (c Config) Validate() error {
	// TODO: add full config validation.
	return errors.Join(c.Log.Validate(), c.HTTP.Validate())
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

	// map ссылочный: копируем, чтобы не затереть DSN в исходном конфиге.
	dbs := make(map[string]pgcollector.Config, len(c.Databases))
	for k, v := range c.Databases {
		v.DSN = "postgres://***"
		dbs[k] = v
	}

	c.Databases = dbs

	// то же для токенов Prometheus/Grafana; BasicAuth — указатель, копируем структуру.
	graf := make(map[string]prometheus.GrafanaConfig, len(c.Grafana))
	for k, v := range c.Grafana {
		v.Token = "***"
		graf[k] = v
	}

	c.Grafana = graf

	prom := make(map[string]prometheus.Config, len(c.Prometheus))
	for k, v := range c.Prometheus {
		v.BearerToken = "***"
		if v.BasicAuth != nil {
			ba := *v.BasicAuth
			ba.Password = "***"
			v.BasicAuth = &ba
		}

		prom[k] = v
	}

	c.Prometheus = prom

	zbx := make(map[string]zabbix.Config, len(c.Zabbix))
	for k, v := range c.Zabbix {
		v.Token = "***"
		zbx[k] = v
	}

	c.Zabbix = zbx

	return slog.AnyValue(safeConfig(c))
}

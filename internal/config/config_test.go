package config_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"

	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	pgcollector "support_bot/internal/collector/postgres"
	"support_bot/internal/collector/prometheus"
	"support_bot/internal/collector/zabbix"
	"support_bot/internal/config"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	maxbot "support_bot/internal/max_bot"
	"support_bot/internal/pkg/retry"
	"support_bot/internal/pkg/secret"
	"support_bot/internal/postgres"
	tgbot "support_bot/internal/tg_bot"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullSecretsConfig() config.Config {
	return config.Config{
		Database: postgres.Config{
			Password: "db-pass-1",
			DSN:      "postgres://u:dsn-pass-2@h/db",
		},
		TgBot:      tgbot.Config{TelegramToken: "tg-token-3"},
		MaxBot:     maxbot.Config{Token: "max-token-4"},
		SMTP:       smtp.Config{Password: "smtp-pass-5"},
		SMB:        smb.Config{Password: "smb-pass-6"},
		Jira:       jira.Config{AuthToken: "jira-token-7"},
		AppMetrica: appmetrica.Config{OAuthToken: "appm-token-8"},
		Databases: map[string]pgcollector.Config{
			"analytics": {DSN: "postgres://ro:map-dsn-9@host:5432/db"},
		},
		Grafana: map[string]prometheus.GrafanaConfig{
			"ops": {Token: "graf-token-10"},
		},
		Prometheus: map[string]prometheus.Config{
			"a": {BearerToken: "bearer-11"},
			"b": {
				BasicAuth: &prometheus.BasicAuth{
					User:     "u",
					Password: "basic-pass-12",
				},
			},
		},
		Zabbix: map[string]zabbix.Config{"z": {Token: "zbx-token-13"}},
	}
}

var allSecrets = []string{
	"db-pass-1",
	"dsn-pass-2",
	"tg-token-3",
	"max-token-4",
	"smtp-pass-5",
	"smb-pass-6",
	"jira-token-7",
	"appm-token-8",
	"map-dsn-9",
	"graf-token-10",
	"bearer-11",
	"basic-pass-12",
	"zbx-token-13",
}

func TestLogDoesNotLeakSecrets(t *testing.T) {
	t.Parallel()

	cfg := fullSecretsConfig()

	for name, newHandler := range map[string]func(*bytes.Buffer) slog.Handler{
		"json": func(b *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(b, nil) },
		"text": func(b *bytes.Buffer) slog.Handler { return slog.NewTextHandler(b, nil) },
	} {
		var buf bytes.Buffer

		slog.New(newHandler(&buf)).Info("cfg", slog.Any("config", cfg))
		require.Contains(t, buf.String(), "***", name)

		for _, s := range allSecrets {
			assert.NotContains(t, buf.String(), s, name)
		}
	}

	assert.Equal(t, "graf-token-10", cfg.Grafana["ops"].Token.Value())
}

// TestSecretFieldsAreSecretString: поля с token|password|dsn|secret в имени — только secret.String.
func TestSecretFieldsAreSecretString(t *testing.T) {
	t.Parallel()

	re := regexp.MustCompile(`(?i)token|password|dsn|secret`)
	// исключения: путь поля -> причина. Пока их нет.
	allowed := map[string]string{}

	var walk func(rt reflect.Type, path string, seen map[reflect.Type]bool)

	walk = func(rt reflect.Type, path string, seen map[reflect.Type]bool) {
		for rt.Kind() == reflect.Pointer || rt.Kind() == reflect.Slice || rt.Kind() == reflect.Map {
			rt = rt.Elem()
		}

		if rt.Kind() != reflect.Struct || seen[rt] {
			return
		}

		seen[rt] = true

		for f := range rt.Fields() {
			p := path + "." + f.Name
			if f.Type.Kind() == reflect.String &&
				f.Type != reflect.TypeFor[secret.String]() &&
				re.MatchString(f.Name) {
				_, ok := allowed[p]
				assert.True(t, ok, "%s must be secret.String", p)
			}

			walk(f.Type, p, seen)
		}
	}

	walk(reflect.TypeFor[config.Config](), "Config", map[reflect.Type]bool{})
}

func loadYAML(t *testing.T, body string) *config.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "c.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)

	return cfg
}

func TestLoadDeliveryDelays(t *testing.T) {
	cfg := loadYAML(t, "delivery:\n  retry:\n    delays: [2s, 3s]\n")
	assert.Equal(
		t,
		retry.Delays{2 * time.Second, 3 * time.Second},
		cfg.Delivery.Retry.Delays,
	)

	cfg = loadYAML(t, "metabase_domain: x\n")
	assert.Equal(
		t,
		retry.Delays{time.Second, 5 * time.Second, 15 * time.Second},
		cfg.Delivery.Retry.Delays,
	)

	cfg = loadYAML(t, "delivery:\n  retry:\n    delays: []\n")
	assert.Empty(t, cfg.Delivery.Retry.Delays)
}

func TestLoadMetricsAddrDefault(t *testing.T) {
	cfg := loadYAML(t, "metabase_domain: x\n")
	assert.Equal(t, "127.0.0.1:9090", cfg.Metrics.Addr)
	assert.False(t, cfg.Metrics.Enabled)

	cfg = loadYAML(t, "metrics:\n  enabled: true\n")
	assert.True(t, cfg.Metrics.Enabled)
}

func TestLoadRegistersDSNSecrets(t *testing.T) {
	dsn := "postgres://u:cfgpw-8842@h:5432/db"
	loadYAML(t, "databases:\n  main:\n    dsn: "+dsn+"\n")

	assert.Equal(t, "x ***", secret.Redact("x cfgpw-8842"))
	assert.Equal(t, "***", secret.Redact(dsn))
}

// validMinimal — дефолтный конфиг без необязательных секций.
func validMinimal() config.Config {
	c := *config.Default()

	c.SMTP, c.SMB, c.Jira, c.MaxBot = smtp.Config{}, smb.Config{}, jira.Config{}, maxbot.Config{}

	return c
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr []string
	}{
		{name: "минимальный конфиг", mutate: func(*config.Config) {}},
		{
			name:    "нет токена telegram",
			mutate:  func(c *config.Config) { c.TgBot.TelegramToken = "" },
			wantErr: []string{"telegram.telegram_token: not set"},
		},
		{
			name:   "нет полей database",
			mutate: func(c *config.Config) { c.Database = postgres.Config{} },
			wantErr: []string{
				"database.host",
				"database.user",
				"database.name",
			},
		},
		{
			name:    "smtp.host без остальных полей",
			mutate:  func(c *config.Config) { c.SMTP.Host = "smtp" },
			wantErr: []string{"smtp.email", "smtp.password", "smtp.port"},
		},
		{
			name:   "smb включён без полей",
			mutate: func(c *config.Config) { c.SMB.Active = true },
			wantErr: []string{
				"smb.address",
				"smb.user",
				"smb.password",
				"smb.share",
			},
		},
		{
			name:   "smb выключен с мусором",
			mutate: func(c *config.Config) { c.SMB = smb.Config{Address: "a"} },
		},
		{
			name:    "max включён без токена",
			mutate:  func(c *config.Config) { c.MaxBot.Enabled = true },
			wantErr: []string{"max.max_token"},
		},
		{
			name:    "jira_host без токена",
			mutate:  func(c *config.Config) { c.Jira.JiraHost = "https://j" },
			wantErr: []string{"jira.auth_token"},
		},
		{
			name: "две ошибки разом",
			mutate: func(c *config.Config) {
				c.TgBot.TelegramToken = ""
				c.SMTP = smtp.Config{Host: "smtp", Port: "25", Password: "p"}
			},
			wantErr: []string{"telegram.telegram_token", "smtp.email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validMinimal()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if len(tt.wantErr) == 0 {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			for _, s := range tt.wantErr {
				assert.ErrorContains(t, err, s)
			}
		})
	}
}

package config_test

import (
	"fmt"
	"support_bot/internal/collector/postgres"
	"support_bot/internal/collector/prometheus"
	"support_bot/internal/collector/zabbix"
	"support_bot/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogValueMasksDatabases(t *testing.T) {
	t.Parallel()

	const dsn = "postgres://ro:secret@host:5432/db" //nolint:gosec // фиктивный DSN для теста

	cfg := config.Config{Databases: map[string]postgres.Config{"analytics": {DSN: dsn}}}

	out := fmt.Sprintf("%+v", cfg.LogValue().Any())

	assert.NotContains(t, out, "secret")
	assert.Equal(t, dsn, cfg.Databases["analytics"].DSN)
}

func TestLogValueMasksPrometheus(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Grafana: map[string]prometheus.GrafanaConfig{"ops": {Token: "tok-secret"}},
		Prometheus: map[string]prometheus.Config{
			"a": {BearerToken: "bearer-secret"},
			"b": {BasicAuth: &prometheus.BasicAuth{User: "u", Password: "pass-secret"}},
		},
		Zabbix: map[string]zabbix.Config{"z": {Token: "zbx-secret"}},
	}

	out := fmt.Sprintf("%+v", cfg.LogValue().Any())

	for _, secret := range []string{"tok-secret", "bearer-secret", "pass-secret", "zbx-secret"} {
		assert.NotContains(t, out, secret)
	}

	assert.Equal(t, "tok-secret", cfg.Grafana["ops"].Token)
	assert.Equal(t, "bearer-secret", cfg.Prometheus["a"].BearerToken)
	assert.Equal(t, "pass-secret", cfg.Prometheus["b"].BasicAuth.Password)
	assert.Equal(t, "zbx-secret", cfg.Zabbix["z"].Token)
}

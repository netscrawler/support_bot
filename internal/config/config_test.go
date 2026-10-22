package config_test

import (
	"fmt"
	"support_bot/internal/collector/postgres"
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

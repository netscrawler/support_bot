package cli

import (
	"support_bot/internal/collector/prometheus"
	"support_bot/internal/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalConfigKeepsRealSecrets(t *testing.T) {
	cfg := config.Default()
	cfg.TgBot.TelegramToken = "tg-real-token"
	cfg.Database.Password = "db-real-pass"
	cfg.Prometheus = map[string]prometheus.Config{"a": {BearerToken: "prom-real-bearer"}}

	for _, format := range []string{"yaml", "env"} {
		out, err := marshalConfig(cfg, format)
		require.NoError(t, err)
		require.Contains(t, out, "tg-real-token", format)
		require.NotContains(t, out, "***", format)

		// env-вывод берёт env-default вместо текущего значения — это не про секреты.
		if format == "yaml" {
			require.Contains(t, out, "db-real-pass")
			require.Contains(t, out, "prom-real-bearer")
		}
	}
}

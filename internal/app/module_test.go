package app

import (
	"log/slog"
	"testing"

	"support_bot/internal/config"

	"go.uber.org/fx"
)

// Граф зависимостей fx собирается без запуска конструкторов.
func TestModuleGraph(t *testing.T) {
	t.Parallel()

	if err := fx.ValidateApp(
		Module,
		fx.Supply(&config.Config{}, slog.Default()),
	); err != nil {
		t.Fatal(err)
	}
}

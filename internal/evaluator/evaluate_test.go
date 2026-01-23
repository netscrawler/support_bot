package evaluator_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"support_bot/internal/evaluator"
)

func TestEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	th := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(th)

	eval, err := evaluator.NewEvaluator(l)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("simple test", func(t *testing.T) {
		t.Parallel()

		report := map[string][]map[string]any{
			"sheet1": {
				{"total": 0, "count": 5},
				{"total": 10, "count": 0},
			},
		}

		fn := `report["sheet1"].all(r, r["total"] != 0)`

		allow, err := eval.Evaluate(t.Context(), report, fn)
		require.NoError(t, err)
		assert.False(t, allow)
	})

	t.Run("count test one sheet", func(t *testing.T) {
		t.Parallel()

		report := map[string][]map[string]any{
			"sheet1": {},
		}

		fn := `size(report["sheet1"]) > 1`

		allow, err := eval.Evaluate(t.Context(), report, fn)
		require.NoError(t, err)
		assert.False(t, allow)
	})

	t.Run("count test all sheets", func(t *testing.T) {
		t.Parallel()

		report := map[string][]map[string]any{
			"sheet1": {},
			"sheet2": {
				{"total": 0, "count": 5},
				{"total": 10, "count": 0},
			},
		}

		fn := `report.map(k, report[k]).flatten().size() > 1`

		allow, err := eval.Evaluate(t.Context(), report, fn)
		require.NoError(t, err)
		assert.True(t, allow)
	})

	t.Run("always expr", func(t *testing.T) {
		t.Parallel()
		ok, err := eval.Evaluate(t.Context(), nil, "[*]")
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = eval.Evaluate(t.Context(), nil, "[!*]")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

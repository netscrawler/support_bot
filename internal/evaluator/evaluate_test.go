package evaluator_test

import (
	"testing"

	"support_bot/internal/evaluator"

	"github.com/stretchr/testify/assert"
)

func TestEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	eval, err := evaluator.NewEvaluator()
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
		assert.NoError(t, err)
		assert.Equal(t, false, allow)
	})
	t.Run("simple test list", func(t *testing.T) {
		t.Parallel()
		report := map[string][][]string{
			"sheet1": {
				{"total", "count"},
				{"0", "5"},
				{"10", "0"},
			},
		}
		fn := `report["sheet1"].slice(1, report["sheet1"].size()).all(r, int(r[0]) != 0)`

		allow, err := eval.EvaluateMatrix(t.Context(), report, fn)
		assert.NoError(t, err)
		assert.Equal(t, false, allow)
	})

	t.Run("always expr", func(t *testing.T) {
		ok, err := eval.Evaluate(t.Context(), nil, "[*]")
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = eval.Evaluate(t.Context(), nil, "[!*]")
		assert.NoError(t, err)
		assert.True(t, !ok)
	})
}

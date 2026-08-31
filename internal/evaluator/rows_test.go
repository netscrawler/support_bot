package evaluator_test

import (
	"support_bot/internal/evaluator"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRowEngine_Match(t *testing.T) {
	t.Parallel()

	eng, err := evaluator.NewRowEngine()
	require.NoError(t, err)

	rows := []map[string]any{
		{"channel": "email", "accepted": 40, "target": 100},
		{"channel": "web", "accepted": 180, "target": 100},
		{"channel": "chat", "accepted": 95, "target": 100},
	}

	t.Run("value comparison", func(t *testing.T) {
		t.Parallel()

		got, err := eng.Match(t.Context(), "value < 100", rows[0], 40, rows)
		require.NoError(t, err)
		assert.True(t, got)

		got, err = eng.Match(t.Context(), "value < 100", rows[1], 180, rows)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("row field", func(t *testing.T) {
		t.Parallel()

		got, err := eng.Match(
			t.Context(),
			`row.channel == "email" && row.accepted < row.target`,
			rows[0], nil, rows,
		)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("aggregate over rows without a max builtin", func(t *testing.T) {
		t.Parallel()

		// "this cell holds the largest accepted value"
		got, err := eng.Match(t.Context(), "rows.all(r, r.accepted <= value)", rows[1], 180, rows)
		require.NoError(t, err)
		assert.True(t, got)

		got, err = eng.Match(t.Context(), "rows.all(r, r.accepted <= value)", rows[0], 40, rows)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("has guard on an absent key yields no match, no error", func(t *testing.T) {
		t.Parallel()

		const expr = "has(row.missing) && row.missing > 5"

		got, err := eng.Match(t.Context(), expr, rows[0], nil, rows)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("unguarded absent key is an error", func(t *testing.T) {
		t.Parallel()

		_, err := eng.Match(t.Context(), "row.missing > 5", rows[0], nil, rows)
		require.Error(t, err)
	})
}

func TestRowEngine_Compile(t *testing.T) {
	t.Parallel()

	eng, err := evaluator.NewRowEngine()
	require.NoError(t, err)

	require.NoError(t, eng.Compile("value < 100", `row.channel == "web"`))

	err = eng.Compile("value <")
	require.Error(t, err)

	// A non-boolean expression compiles but Match rejects its result.
	require.NoError(t, eng.Compile("value + 1"))

	_, err = eng.Match(t.Context(), "value + 1", map[string]any{}, 3, nil)
	require.Error(t, err)
}

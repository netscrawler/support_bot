package core_test

import (
	"support_bot/internal/core"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataFromMapSlice(t *testing.T) {
	t.Parallel()
	t.Run("one table load", func(t *testing.T) {
		t.Parallel()

		duck, err := core.NewDB()
		require.NoError(t, err)

		defer func() {
			err := duck.Close()
			require.NoError(t, err)
		}()

		data := map[string][]map[string]any{
			"some_table": {
				{
					"id":   int32(10),
					"name": "john",
				},
				{
					"id":   int32(11),
					"name": "jhoe",
				},
				{
					"id":   int32(12),
					"name": "petr",
				},
			},
		}

		err = duck.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duck.ExecuteQuery(t.Context(), "SELECT * FROM some_table")
		require.NoError(t, err)
		assert.Equal(t, data["some_table"], res)
	})

	t.Run("two table load", func(t *testing.T) {
		t.Parallel()

		duck, err := core.NewDB()
		require.NoError(t, err)

		defer func() {
			err := duck.Close()
			require.NoError(t, err)
		}()

		data := map[string][]map[string]any{
			"some_table": {
				{
					"id":   10,
					"name": "john",
				},
				{
					"id":   11,
					"name": "jhoe",
				},
				{
					"id":   12,
					"name": "petr",
				},
			},
			"other_table": {
				{
					"id":       10,
					"vacation": "it",
				},
				{
					"id":       11,
					"vacation": "it",
				},
				{
					"id":       12,
					"vacation": "svarshik",
				},
			},
		}

		err = duck.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duck.ExecuteQuery(t.Context(), `SELECT some_table.id, name, other_table.vacation
			FROM some_table 
left join other_table on some_table.id = other_table.id`)
		require.NoError(t, err)

		expected := []map[string]any{
			{
				"id":       int32(10),
				"name":     "john",
				"vacation": "it",
			},
			{
				"id":       int32(11),
				"name":     "jhoe",
				"vacation": "it",
			},
			{
				"id":       int32(12),
				"name":     "petr",
				"vacation": "svarshik",
			},
		}
		assert.Equal(t, expected, res)
	})

	t.Run("one table not determine schema", func(t *testing.T) {
		t.Parallel()

		duck, err := core.NewDB()
		require.NoError(t, err)

		defer func() {
			err := duck.Close()
			require.NoError(t, err)
		}()

		data := map[string][]map[string]any{
			"some_table": {
				{
					"id":   10,
					"name": "john",
				},
				{
					"id":      11,
					"name":    "jhoe",
					"surname": "doe",
				},
				{
					"id":       12,
					"name":     "petr",
					"vacation": "it",
				},
			},
		}

		err = duck.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duck.ExecuteQuery(t.Context(), "SELECT * FROM some_table where id = 12")
		require.NoError(t, err)

		expected := []map[string]any{
			{
				"id":       int32(12),
				"name":     "petr",
				"surname":  nil,
				"vacation": "it",
			},
		}
		assert.Equal(t, expected, res)
	})
}

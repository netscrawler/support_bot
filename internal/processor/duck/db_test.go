//go:build duckdb

package duck_test

import (
	"support_bot/internal/processor/duck"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataFromMapSlice(t *testing.T) {
	t.Parallel()
	t.Run("one table load", func(t *testing.T) {
		t.Parallel()

		duckDB, err := duck.New()
		require.NoError(t, err)

		defer func() {
			err := duckDB.Close()
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

		err = duckDB.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duckDB.ExecuteQuery(t.Context(), "SELECT * FROM some_table")
		require.NoError(t, err)
		assert.Equal(t, data["some_table"], res)
	})

	t.Run("two table load", func(t *testing.T) {
		t.Parallel()

		duckDB, err := duck.New()
		require.NoError(t, err)

		defer func() {
			err := duckDB.Close()
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

		err = duckDB.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duckDB.ExecuteQuery(
			t.Context(),
			`SELECT some_table.id, name, other_table.vacation
			FROM some_table 
left join other_table on some_table.id = other_table.id`,
		)
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

		duckDB, err := duck.New()
		require.NoError(t, err)

		defer func() {
			err := duckDB.Close()
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

		err = duckDB.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duckDB.ExecuteQuery(t.Context(), "SELECT * FROM some_table where id = 12")
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

	t.Run("grouping op", func(t *testing.T) {
		t.Parallel()

		duckDB, err := duck.New()
		require.NoError(t, err)

		defer func() {
			err := duckDB.Close()
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

		err = duckDB.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duckDB.ExecuteQuery(t.Context(), "SELECT count()as count FROM some_table")
		require.NoError(t, err)
		assert.Equal(t, []map[string]any{{"count": int64(3)}}, res)
	})
}

func TestSqlOperations(t *testing.T) {
	t.Parallel()
	t.Run("join tables", func(t *testing.T) {
		t.Parallel()

		duckDB, err := duck.New()
		require.NoError(t, err)

		defer func() {
			err := duckDB.Close()
			require.NoError(t, err)
		}()

		data := map[string][]map[string]any{
			"users": {
				{
					"id":       10,
					"name":     "john",
					"vacation": 1,
				},
				{
					"id":       11,
					"name":     "jhoe",
					"surname":  "doe",
					"vacation": 2,
				},
				{
					"id":       12,
					"name":     "petr",
					"vacation": 3,
				},
			},
			"vacations": {
				{
					"id":   1,
					"name": "it",
				},
				{
					"id":   2,
					"name": "manager",
				},
				{
					"id":   3,
					"name": "ceo",
				},
			},
		}

		err = duckDB.LoadDataFromMapSlice(t.Context(), data)
		require.NoError(t, err)

		res, err := duckDB.ExecuteQuery(
			t.Context(),
			"SELECT users.name, vacations.name as vacation_name FROM users JOIN vacations ON users.vacation = vacations.id",
		)
		require.NoError(t, err)
		expected := []map[string]any{
			{
				"name":          "john",
				"vacation_name": "it",
			},
			{
				"name":          "jhoe",
				"vacation_name": "manager",
			},
			{
				"name":          "petr",
				"vacation_name": "ceo",
			},
		}
		assert.Equal(t, expected, res)
	})
}

func TestNoRowsInResult(t *testing.T) {
	t.Parallel()

	duckDB, err := duck.New()
	require.NoError(t, err)

	defer func() {
		err := duckDB.Close()
		require.NoError(t, err)
	}()

	data := map[string][]map[string]any{
		"users":     {},
		"vacations": {},
	}

	err = duckDB.LoadDataFromMapSlice(t.Context(), data)
	require.NoError(t, err)

	res, err := duckDB.ExecuteQuery(
		t.Context(),
		"SELECT users.name, vacations.name as vacation_name FROM users JOIN vacations ON users.vacation = vacations.id",
	)
	require.NoError(t, err)
	t.Log(res)
}

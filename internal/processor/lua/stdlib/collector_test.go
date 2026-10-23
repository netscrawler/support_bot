package stdlib_test

import (
	"context"
	"errors"
	"support_bot/internal/processor/lua/stdlib"
	"testing"

	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

type fakePG struct {
	rows   []map[string]any
	err    error
	db     string
	sql    string
	params map[string]string
}

func (f *fakePG) Fetch(
	_ context.Context,
	db, sql string,
	params map[string]string,
) ([]map[string]any, error) {
	f.db, f.sql, f.params = db, sql, params

	return f.rows, f.err
}

func run(t *testing.T, queries map[string]stdlib.QueryCollector, script string) *lua.LState {
	t.Helper()

	L := lua.NewState()
	t.Cleanup(L.Close)
	stdlib.NewCollector(nil, queries).Register(L)
	require.NoError(t, L.DoString(script))

	return L
}

func TestPGCollect(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		f := &fakePG{rows: []map[string]any{{"x": float64(5)}}}
		L := run(t, map[string]stdlib.QueryCollector{"pg": f}, `
local rows, err = collector.pg.collect("analytics", "select @x::int as x", {x = "5"})
x, e = rows[1].x, err`)

		require.Equal(t, "analytics", f.db)
		require.Equal(t, "select @x::int as x", f.sql)
		require.Equal(t, map[string]string{"x": "5"}, f.params)
		require.Equal(t, lua.LNumber(5), L.GetGlobal("x"))
		require.Equal(t, lua.LNil, L.GetGlobal("e"))
	})

	t.Run("error", func(t *testing.T) {
		L := run(
			t,
			map[string]stdlib.QueryCollector{"pg": &fakePG{err: errors.New("boom")}},
			`rows, err = collector.pg.collect("a", "select 1")`,
		)

		require.Equal(t, lua.LNil, L.GetGlobal("rows"))
		require.Equal(t, lua.LString("boom"), L.GetGlobal("err"))
	})

	t.Run("absent", func(t *testing.T) {
		L := run(t, nil, `p = collector.pg; q = collector.prom`)

		require.Equal(t, lua.LNil, L.GetGlobal("p"))
		require.Equal(t, lua.LNil, L.GetGlobal("q"))
	})
}

func TestPromCollect(t *testing.T) {
	t.Parallel()

	f := &fakePG{rows: []map[string]any{{"v": float64(1)}}}
	q := map[string]stdlib.QueryCollector{"prom": f}
	L := run(t, q, `rows, err = collector.prom.collect("main", "up", {k = "v"})`)

	require.Equal(t, "main", f.db)
	require.Equal(t, "up", f.sql)
	require.Equal(t, map[string]string{"k": "v"}, f.params)
	require.Equal(t, lua.LNil, L.GetGlobal("err"))

	f.err = errors.New("boom")
	L = run(t, q, `rows, err = collector.prom.collect("main", "up")`)

	require.Equal(t, lua.LNil, L.GetGlobal("rows"))
	require.Equal(t, lua.LString("boom"), L.GetGlobal("err"))
}

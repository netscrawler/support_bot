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

func run(t *testing.T, pg stdlib.SQLCollector, script string) *lua.LState {
	t.Helper()

	L := lua.NewState()
	t.Cleanup(L.Close)
	stdlib.NewCollector(nil, pg).Register(L)
	require.NoError(t, L.DoString(script))

	return L
}

func TestPGCollect(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		f := &fakePG{rows: []map[string]any{{"x": float64(5)}}}
		L := run(t, f, `
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
			&fakePG{err: errors.New("boom")},
			`rows, err = collector.pg.collect("a", "select 1")`,
		)

		require.Equal(t, lua.LNil, L.GetGlobal("rows"))
		require.Equal(t, lua.LString("boom"), L.GetGlobal("err"))
	})

	t.Run("absent", func(t *testing.T) {
		L := run(t, nil, `p = collector.pg`)

		require.Equal(t, lua.LNil, L.GetGlobal("p"))
	})
}

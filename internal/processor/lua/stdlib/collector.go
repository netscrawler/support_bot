package stdlib

import (
	"context"
	"time"

	luapkg "support_bot/internal/pkg/lua"

	lua "github.com/yuin/gopher-lua"
)

type DirectCollector interface {
	Fetch(ctx context.Context, target string, params map[string]string) ([]map[string]any, error)
}

// SQLCollector выполняет SQL только на чтение в именованной базе.
type SQLCollector interface {
	Fetch(ctx context.Context, db, sql string, params map[string]string) ([]map[string]any, error)
}

type CollectPlugin struct {
	directs map[string]DirectCollector
	pg      SQLCollector
}

// pg без базы — настоящий nil-интерфейс, не типизированный nil-указатель.
func NewCollector(directs map[string]DirectCollector, pg SQLCollector) *CollectPlugin {
	if directs == nil {
		directs = make(map[string]DirectCollector)
	}
	return &CollectPlugin{directs: directs, pg: pg}
}

func (c *CollectPlugin) Register(L *lua.LState) {
	root := L.NewTable()

	for name := range c.directs {
		mod := L.NewTable()

		L.SetFuncs(mod, map[string]lua.LGFunction{
			"collect": c.luaCollect(name),
		})

		root.RawSetString(name, mod)
	}

	if c.pg != nil {
		mod := L.NewTable()
		L.SetFuncs(mod, map[string]lua.LGFunction{"collect": c.luaPGCollect})
		root.RawSetString("pg", mod)
	}

	L.SetGlobal("collector", root)
}

func (c *CollectPlugin) luaCollect(name string) lua.LGFunction {
	return func(L *lua.LState) int {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		collector := c.directs[name]
		if collector == nil {
			L.Push(lua.LNil)
			L.Push(lua.LString("collector not found"))

			return 2
		}

		target := L.CheckString(1)

		rows, err := collector.Fetch(ctx, target, luaParams(L, 2))

		return pushRows(L, rows, err)
	}
}

func (c *CollectPlugin) luaPGCollect(l *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db := l.CheckString(1)
	sql := l.CheckString(2)

	rows, err := c.pg.Fetch(ctx, db, sql, luaParams(l, 3))

	return pushRows(l, rows, err)
}

// luaParams читает необязательную таблицу-аргумент idx в map строк.
func luaParams(l *lua.LState, idx int) map[string]string {
	params := map[string]string{}
	if l.GetTop() >= idx {
		l.CheckTable(idx).ForEach(func(k, v lua.LValue) {
			params[lua.LVAsString(k)] = lua.LVAsString(v)
		})
	}

	return params
}

func pushRows(l *lua.LState, rows []map[string]any, err error) int {
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(err.Error()))

		return 2
	}

	l.Push(rowsToLua(l, rows))
	l.Push(lua.LNil)

	return 2
}

func rowsToLua(
	L *lua.LState,
	rows []map[string]any,
) *lua.LTable {
	arr := L.NewTable()

	for _, row := range rows {
		rowTbl := L.NewTable()

		for k, v := range row {
			rowTbl.RawSetString(k, luapkg.GoValueToLua(L, v))
		}

		arr.Append(rowTbl)
	}

	return arr
}

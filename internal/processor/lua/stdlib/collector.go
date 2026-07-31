package stdlib

import (
	"context"
	"time"

	"support_bot/internal/pkg/lua"

	lua "github.com/yuin/gopher-lua"
)

type DirectCollector interface {
	Fetch(ctx context.Context, target string, params map[string]string) ([]map[string]any, error)
}

type CollectPlugin struct {
	directs map[string]DirectCollector
}

func NewCollector(directs map[string]DirectCollector) *CollectPlugin {
	if directs == nil {
		directs = make(map[string]DirectCollector)
	}
	return &CollectPlugin{directs: directs}
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

		params := map[string]string{}
		if L.GetTop() >= 2 {
			tbl := L.CheckTable(2)

			tbl.ForEach(func(k, v lua.LValue) {
				params[lua.LVAsString(k)] = lua.LVAsString(v)
			})
		}

		rows, err := collector.Fetch(ctx, target, params)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))

			return 2
		}

		L.Push(rowsToLua(L, rows))
		L.Push(lua.LNil)

		return 2
	}
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

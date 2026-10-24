package stdlib

import (
	"context"
	"log/slog"

	luapkg "support_bot/internal/pkg/lua"
	"support_bot/internal/processor/duck"

	lua "github.com/yuin/gopher-lua"
)

type DatabasePlugin struct{}

type LuaDatabase struct {
	DB *duck.DB
}

func (d DatabasePlugin) Register(l *lua.LState) {
	mt := l.NewTypeMetatable("DuckDB")

	l.SetField(mt, "__index",
		l.SetFuncs(l.NewTable(), map[string]lua.LGFunction{
			"query": d.luaQuery,
			// "exec":     d.luaExec,
			"loadData": d.luaLoadData,
			"close":    d.luaClose,
		}),
	)
	l.SetField(mt, "__gc", l.NewFunction(d.luaGC))

	mod := l.NewTable()

	l.SetFuncs(mod, map[string]lua.LGFunction{
		"new": d.luaNew,
	})

	l.SetGlobal("duck", mod)
}

func (d DatabasePlugin) luaGC(l *lua.LState) int {
	db := checkDB(l)

	if db.DB != nil {
		if err := db.DB.Close(); err != nil {
			slog.Error("close Lua DuckDB", slog.Any("error", err))
		}

		db.DB = nil
	}

	return 0
}

func (d DatabasePlugin) luaNew(l *lua.LState) int {
	db, err := duck.New()
	if err != nil {
		l.RaiseError("%s", err.Error())

		return 0
	}

	ud := l.NewUserData()

	ud.Value = &LuaDatabase{
		DB: db,
	}

	l.SetMetatable(
		ud,
		l.GetTypeMetatable("DuckDB"),
	)

	l.Push(ud)

	return 1
}

func checkDB(l *lua.LState) *LuaDatabase {
	ud := l.CheckUserData(1)

	db, ok := ud.Value.(*LuaDatabase)
	if !ok {
		l.ArgError(1, "DuckDB expected")

		return nil
	}

	return db
}

// luaExecuteQuery executes a SQL query and returns results
// Usage: results, err = stdlib.ExecuteQuery(query_string)
// Returns: table of results (or nil), error message (or nil).
func (d DatabasePlugin) luaQuery(l *lua.LState) int {
	db := checkDB(l)

	sql := l.CheckString(2)

	ctx := context.Background()

	rows, err := db.DB.ExecuteQuery(ctx, sql)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(err.Error()))

		return 2
	}

	l.Push(queryResultToLua(l, rows))
	l.Push(lua.LNil)

	return 2
}

func (d DatabasePlugin) luaLoadData(l *lua.LState) int {
	db := checkDB(l)
	ctx := context.TODO()

	tbl := l.CheckTable(2)

	data, err := luapkg.LuaTableToGoData(tbl)
	if err != nil {
		l.Push(lua.LString(err.Error()))

		return 1
	}

	err = db.DB.LoadDataFromMapSlice(ctx, data)
	if err != nil {
		l.Push(lua.LString(err.Error()))

		return 1
	}

	l.Push(lua.LNil)

	return 1
}

func (d DatabasePlugin) luaClose(l *lua.LState) int {
	db := checkDB(l)

	if db.DB != nil {
		if err := db.DB.Close(); err != nil {
			slog.Error("close Lua DuckDB", slog.Any("error", err))
		}

		db.DB = nil
	}

	return 0
}

// queryResultToLua converts query results to Lua table format.
func queryResultToLua(l *lua.LState, results []map[string]any) *lua.LTable {
	arr := l.NewTable()

	for _, row := range results {
		rowTbl := l.NewTable()

		for k, v := range row {
			rowTbl.RawSetString(k, luapkg.GoValueToLua(l, v))
		}

		arr.Append(rowTbl)
	}

	return arr
}

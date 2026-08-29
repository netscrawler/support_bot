package stdlib

import (
	"context"
	"support_bot/internal/processor/duck"

	luapkg "support_bot/internal/pkg/lua"

	lua "github.com/yuin/gopher-lua"
)

const duckDbType = "DuckDB"

type DatabasePlugin struct{}

type LuaDatabase struct {
	DB *duck.DB
}

func (d DatabasePlugin) Register(L *lua.LState) {
	mt := L.NewTypeMetatable("DuckDB")

	L.SetField(mt, "__index",
		L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"query": d.luaQuery,
			// "exec":     d.luaExec,
			"loadData": d.luaLoadData,
			"close":    d.luaClose,
		}),
	)
	L.SetField(mt, "__gc", L.NewFunction(d.luaGC))

	mod := L.NewTable()

	L.SetFuncs(mod, map[string]lua.LGFunction{
		"new": d.luaNew,
	})

	L.SetGlobal("duck", mod)
}

func (d *DatabasePlugin) luaGC(L *lua.LState) int {
	db := checkDB(L)

	if db.DB != nil {
		_ = db.DB.Close()
		db.DB = nil
	}

	return 0
}

func (d *DatabasePlugin) luaNew(L *lua.LState) int {
	db, err := duck.New()
	if err != nil {
		L.RaiseError("%s", err.Error())
		return 0
	}

	ud := L.NewUserData()
	ud.Value = &LuaDatabase{
		DB: db,
	}

	L.SetMetatable(
		ud,
		L.GetTypeMetatable("DuckDB"),
	)

	L.Push(ud)

	return 1
}

func checkDB(L *lua.LState) *LuaDatabase {
	ud := L.CheckUserData(1)

	db, ok := ud.Value.(*LuaDatabase)
	if !ok {
		L.ArgError(1, "DuckDB expected")
		return nil
	}

	return db
}

// luaExecuteQuery executes a SQL query and returns results
// Usage: results, err = stdlib.ExecuteQuery(query_string)
// Returns: table of results (or nil), error message (or nil).
func (d *DatabasePlugin) luaQuery(L *lua.LState) int {
	db := checkDB(L)

	sql := L.CheckString(2)

	ctx := context.Background()

	rows, err := db.DB.ExecuteQuery(ctx, sql)
	if err != nil {

		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))

		return 2
	}

	L.Push(queryResultToLua(L, rows))
	L.Push(lua.LNil)

	return 2
}

func (d *DatabasePlugin) luaLoadData(L *lua.LState) int {
	db := checkDB(L)
	ctx := context.TODO()

	tbl := L.CheckTable(2)

	data, err := luapkg.LuaTableToGoData(tbl)
	if err != nil {

		L.Push(lua.LString(err.Error()))

		return 1
	}

	err = db.DB.LoadDataFromMapSlice(ctx, data)
	if err != nil {

		L.Push(lua.LString(err.Error()))

		return 1
	}

	L.Push(lua.LNil)

	return 1
}

func (d *DatabasePlugin) luaClose(L *lua.LState) int {
	db := checkDB(L)

	if db ==nil{
		return 0
	}

	if db.DB != nil {
		_ = db.DB.Close()
		db.DB = nil
	}

	return 0
}
// queryResultToLua converts query results to Lua table format.
func queryResultToLua(L *lua.LState, results []map[string]any) *lua.LTable {
	arr := L.NewTable()

	for _, row := range results {
		rowTbl := L.NewTable()

		for k, v := range row {
			rowTbl.RawSetString(k, luapkg.GoValueToLua(L, v))
		}

		arr.Append(rowTbl)
	}

	return arr
}

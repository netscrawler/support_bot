package stdlib

import (
	"context"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// DatabaseFunc interface wraps the core DB functionality.
type DatabaseFunc interface {
	ExecuteQuery(ctx context.Context, query string) ([]map[string]any, error)
	LoadDataFromMapSlice(ctx context.Context, sample map[string][]map[string]any) error
}

type DatabasePlugin struct {
	db DatabaseFunc
}

func NewDatabase(db DatabaseFunc) *DatabasePlugin {
	return &DatabasePlugin{db: db}
}

// luaExecuteQuery executes a SQL query and returns results
// Usage: results, err = stdlib.ExecuteQuery(query_string)
// Returns: table of results (or nil), error message (or nil).
func (d *DatabasePlugin) luaExecuteQuery(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	query := L.CheckString(1)
	if query == "" {
		L.Push(lua.LNil)
		L.Push(lua.LString("query cannot be empty"))

		return 2
	}

	results, err := d.db.ExecuteQuery(ctx, query)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))

		return 2
	}

	L.Push(queryResultToLua(L, results))
	L.Push(lua.LNil)

	return 2
}

// luaLoadData loads data from a map structure into the database
// Usage: err = stdlib.LoadData(data_table)
// Returns: error message (or nil) if successful.
func (d *DatabasePlugin) luaLoadData(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dataTable := L.CheckTable(1)

	data, err := luaTableToGoData(dataTable)
	if err != nil {
		L.Push(lua.LString(err.Error()))

		return 1
	}

	err = d.db.LoadDataFromMapSlice(ctx, data)
	if err != nil {
		L.Push(lua.LString(err.Error()))

		return 1
	}

	L.Push(lua.LNil)

	return 1
}

// queryResultToLua converts query results to Lua table format.
func queryResultToLua(L *lua.LState, results []map[string]any) *lua.LTable {
	arr := L.NewTable()

	for _, row := range results {
		rowTbl := L.NewTable()

		for k, v := range row {
			rowTbl.RawSetString(k, goValueToLua(L, v))
		}

		arr.Append(rowTbl)
	}

	return arr
}

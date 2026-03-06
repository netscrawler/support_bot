package stdlib_test

import (
	"context"
	"errors"
	"support_bot/internal/plugin/stdlib"
	"testing"

	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

// MockDatabase implements DatabaseFunc interface.
type MockDatabase struct {
	data    map[string][]map[string]any
	lastErr error
}

func (m *MockDatabase) LoadDataFromMapSlice(ctx context.Context, sample map[string][]map[string]any) error {
	if m.lastErr != nil {
		return m.lastErr
	}

	m.data = sample

	return nil
}

func (m *MockDatabase) ExecuteQuery(ctx context.Context, query string) ([]map[string]any, error) {
	if m.lastErr != nil {
		return nil, m.lastErr
	}
	// Simple mock: return all data from first table if query is "SELECT *"
	for _, rows := range m.data {
		return rows, nil
	}

	return []map[string]any{}, nil
}

func TestDatabaseLoadData(t *testing.T) {
	t.Parallel()

	t.Run("successful data load", func(t *testing.T) {
		t.Parallel()

		mockDB := &MockDatabase{}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{DatabasePlugin: stdlib.NewDatabase(mockDB)}
		std.Register(L)

		err := L.DoString(`
			loadErr = stdlib.LoadData({
				users = {
					{ id = 1, name = "Alice" },
				}
			})
		`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("loadErr")
		require.Equal(t, lua.LNil, luaErr)

		require.NotNil(t, mockDB.data)
		require.Equal(t, map[string][]map[string]any{
			"users": {
				{"id": float64(1), "name": "Alice"},
			},
		}, mockDB.data)
	})

	t.Run("load data error", func(t *testing.T) {
		t.Parallel()

		mockDB := &MockDatabase{lastErr: errors.New("load failed")}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{DatabasePlugin: stdlib.NewDatabase(mockDB)}
		std.Register(L)

		err := L.DoString(`
			loadErr = stdlib.LoadData({ users = {} })
		`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("loadErr")
		require.Equal(t, lua.LString("load failed"), luaErr)
	})
}

func TestDatabaseExecuteQuery(t *testing.T) {
	t.Parallel()

	t.Run("successful query execution", func(t *testing.T) {
		t.Parallel()

		mockDB := &MockDatabase{
			data: map[string][]map[string]any{
				"users": {
					{"id": float64(1), "name": "Alice", "age": float64(30)},
					{"id": float64(2), "name": "Bob", "age": float64(25)},
				},
			},
		}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{DatabasePlugin: stdlib.NewDatabase(mockDB)}
		std.Register(L)

		err := L.DoString(`
			result, err = stdlib.ExecuteQuery("SELECT * FROM users")
		`)
		require.NoError(t, err)

		errVal := L.GetGlobal("err")
		resultsVal := L.GetGlobal("result")

		require.Equal(t, lua.LNil, errVal)
		require.IsType(t, &lua.LTable{}, resultsVal)

		goRows := dbLuaRowsToGo(resultsVal.(*lua.LTable))
		require.Equal(t, []map[string]any{
			{"id": float64(1), "name": "Alice", "age": float64(30)},
			{"id": float64(2), "name": "Bob", "age": float64(25)},
		}, goRows)
	})

	t.Run("query with empty string", func(t *testing.T) {
		t.Parallel()

		mockDB := &MockDatabase{}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{DatabasePlugin: stdlib.NewDatabase(mockDB)}
		std.Register(L)

		err := L.DoString(`
			result, err = stdlib.ExecuteQuery("")
		`)
		require.NoError(t, err)

		errVal := L.GetGlobal("err")
		resultsVal := L.GetGlobal("result")

		require.Equal(t, lua.LNil, resultsVal)
		require.Equal(t, lua.LString("query cannot be empty"), errVal)
	})

	t.Run("query with backend error", func(t *testing.T) {
		t.Parallel()

		mockDB := &MockDatabase{lastErr: errors.New("query failed")}

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{DatabasePlugin: stdlib.NewDatabase(mockDB)}
		std.Register(L)

		err := L.DoString(`
			result, err = stdlib.ExecuteQuery("SELECT * FROM users")
		`)
		require.NoError(t, err)

		errVal := L.GetGlobal("err")
		resultsVal := L.GetGlobal("result")

		require.Equal(t, lua.LNil, resultsVal)
		require.Equal(t, lua.LString("query failed"), errVal)
	})
}

func TestDatabaseIntegration(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	mockDB := &MockDatabase{}
	std := stdlib.STD{DatabasePlugin: stdlib.NewDatabase(mockDB)}
	std.Register(L)

	t.Run("load and query workflow", func(t *testing.T) {
		err := L.DoString(`
			loadErr = stdlib.LoadData({
				products = {
					{ id = 1, name = "Laptop", price = 999.99 },
					{ id = 2, name = "Mouse", price = 29.99 },
				}
			})

			result, err = stdlib.ExecuteQuery("SELECT * FROM products")
		`)
		require.NoError(t, err)

		loadErrVal := L.GetGlobal("loadErr")
		errVal := L.GetGlobal("err")
		resultsVal := L.GetGlobal("result")

		require.Equal(t, lua.LNil, loadErrVal)
		require.Equal(t, lua.LNil, errVal)
		require.IsType(t, &lua.LTable{}, resultsVal)

		goRows := dbLuaRowsToGo(resultsVal.(*lua.LTable))
		require.Equal(t, []map[string]any{
			{"id": float64(1), "name": "Laptop", "price": float64(999.99)},
			{"id": float64(2), "name": "Mouse", "price": float64(29.99)},
		}, goRows)
	})
}

func dbLuaValueToGo(v lua.LValue) any {
	switch x := v.(type) {
	case lua.LString:
		return string(x)
	case lua.LNumber:
		return float64(x)
	case lua.LBool:
		return bool(x)
	case *lua.LNilType:
		return nil
	default:
		return nil
	}
}

func dbLuaRowsToGo(tbl *lua.LTable) []map[string]any {
	rows := make([]map[string]any, 0, tbl.Len())

	tbl.ForEach(func(_ lua.LValue, row lua.LValue) {
		rowTbl, ok := row.(*lua.LTable)
		if !ok {
			return
		}

		goRow := make(map[string]any)

		rowTbl.ForEach(func(k, v lua.LValue) {
			goRow[k.String()] = dbLuaValueToGo(v)
		})

		rows = append(rows, goRow)
	})

	return rows
}

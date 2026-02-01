package stdlib_test

import (
	"errors"
	"testing"

	models "support_bot/internal/models/report"
	"support_bot/internal/plugin/stdlib"
	pmock "support_bot/internal/plugin/stdlib/mock"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestCollector(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		mockCollector := &pmock.MockCollector{}

		cards := []models.Card{
			{
				CardUUID: "1afb380d-bc34-49c6-a44a-a2232602e25d",
				Title:    "Card1",
			},
		}

		expected := map[string][]map[string]any{
			"Card1": {
				{"field": "value1"},
			},
		}

		mockCollector.
			On("Collect", mock.Anything, cards).
			Return(expected, nil).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{CollectPlugin: stdlib.NewCollector(mockCollector)}

		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.Collect({
		  {
		    card_uuid = "1afb380d-bc34-49c6-a44a-a2232602e25d",
		    title = "Card1",
		  },
		})
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.IsType(t, &lua.LTable{}, luaResult)

		goResult := luaTableToGo(luaResult.(*lua.LTable))

		require.Equal(t, expected, goResult)

		mockCollector.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		mockCollector := &pmock.MockCollector{}

		cards := []models.Card{
			{
				CardUUID: "1afb380d-bc34-49c6-a44a-a2232602e25d",
				Title:    "Card1",
			},
		}

		wantErr := errors.New("some error")

		mockCollector.
			On("Collect", mock.Anything, cards).
			Return(nil, wantErr).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{CollectPlugin: stdlib.NewCollector(mockCollector)}

		std.Register(L)

		err := L.DoString(`
		result, err = stdlib.Collect({
		  {
		    card_uuid = "1afb380d-bc34-49c6-a44a-a2232602e25d",
		    title = "Card1",
		  }
		})
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LString("some error"), luaErr)

		luaResult := L.GetGlobal("result")

		goResult := luaValueToGo(luaResult)

		require.Equal(t, nil, goResult)

		mockCollector.AssertExpectations(t)
	})
}

func luaValueToGo(v lua.LValue) any {
	switch x := v.(type) {
	case lua.LString:
		return string(x)
	case lua.LNumber:
		return float64(x)
	case lua.LBool:
		return bool(x)
	case *lua.LTable:
		return luaTableToGo(x)
	case *lua.LNilType:
		return nil
	default:
		return nil
	}
}

func luaTableToGo(t *lua.LTable) map[string][]map[string]any {
	out := make(map[string][]map[string]any)

	t.ForEach(func(k, v lua.LValue) {
		key := k.String()
		arr := v.(*lua.LTable)

		var rows []map[string]any

		arr.ForEach(func(_, row lua.LValue) {
			rowTbl := row.(*lua.LTable)
			m := make(map[string]any)

			rowTbl.ForEach(func(fk, fv lua.LValue) {
				m[fk.String()] = luaValueToGo(fv)
			})

			rows = append(rows, m)
		})

		out[key] = rows
	})

	return out
}

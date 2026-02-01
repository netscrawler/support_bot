package stdlib_test

import (
	"errors"
	"testing"

	"support_bot/internal/plugin/stdlib"
	pmock "support_bot/internal/plugin/stdlib/mock"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestEvaluator(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		mockEvaluator := &pmock.MockEvaluator{}

		expected := true
		mockEvaluator.
			On("Evaluate", mock.Anything, mock.Anything, mock.Anything).
			Return(expected, nil).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{EvaluatorPlugin: stdlib.NewEvaluator(mockEvaluator)}

		std.Register(L)

		err := L.DoString(`
	result, err = stdlib.Evaluate(
  {
    Card1 = {
      { field1 = 0 }
    }
  },
  "count(Card1) > 0"
)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, lua.LNil, luaErr)

		luaResult := L.GetGlobal("result")
		require.Equal(t, true, lua.LVAsBool(luaResult))

		goResult := luaValueToGo(luaResult)

		require.Equal(t, expected, goResult)

		mockEvaluator.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		mockEvaluator := &pmock.MockEvaluator{}

		expected := false
		mockEvaluator.
			On("Evaluate", mock.Anything, mock.Anything, mock.Anything).
			Return(expected, errors.New("some error")).
			Once()

		L := lua.NewState()
		defer L.Close()

		std := stdlib.STD{EvaluatorPlugin: stdlib.NewEvaluator(mockEvaluator)}

		std.Register(L)

		err := L.DoString(`
	result, err = stdlib.Evaluate(
  {
    Card1 = {
      { field1 = 0 }
    }
  },
  "count(Card1) > 0"
)
	`)
		require.NoError(t, err)

		luaErr := L.GetGlobal("err")
		require.Equal(t, "some error", luaErr.String())

		luaResult := L.GetGlobal("result")
		require.Equal(t, false, lua.LVAsBool(luaResult))

		goResult := luaValueToGo(luaResult)

		require.Equal(t, nil, goResult)

		mockEvaluator.AssertExpectations(t)
	})
}

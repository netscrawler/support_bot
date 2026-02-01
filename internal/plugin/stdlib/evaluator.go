package stdlib

import (
	"context"
	"time"

	lua "github.com/yuin/gopher-lua"
)

type Evaluator interface {
	Evaluate(
		ctx context.Context,
		data map[string][]map[string]any,
		expr string,
	) (bool, error)
}

type EvaluatorPlugin struct {
	eval Evaluator
}

func NewEvaluator(eval Evaluator) *EvaluatorPlugin {
	return &EvaluatorPlugin{eval: eval}
}

func (p *EvaluatorPlugin) luaEvaluate(L *lua.LState) int {
	// --- ctx ---
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// --- args ---
	luaData := L.CheckTable(1)
	expr := L.CheckString(2)

	// --- Lua → Go ---
	data, err := luaTableToGoData(luaData)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// --- вызов evaluator ---
	ok, evalErr := p.eval.Evaluate(ctx, data, expr)
	if evalErr != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(evalErr.Error()))
		return 2
	}

	// --- Go → Lua ---
	L.Push(lua.LBool(ok))
	L.Push(lua.LNil)
	return 2
}

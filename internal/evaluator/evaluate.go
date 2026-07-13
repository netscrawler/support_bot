package evaluator

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	lru "github.com/hashicorp/golang-lru/v2"
)

type evaluator struct {
	env   *cel.Env
	cache *lru.Cache[string, cel.Program]
}

func NewEvaluator() (*evaluator, error) {
	lT1, err := lru.New[string, cel.Program](3)
	if err != nil {
		return nil, fmt.Errorf("unable create cache: (%w)", err)
	}

	envT1, err := cel.NewEnv(
		cel.StdLib(),
		ext.Lists(),
		ext.Sets(),
		ext.TwoVarComprehensions(),
		cel.OptionalTypes(),
		cel.Macros(cel.StandardMacros...),
		cel.Variable(
			"report",
			cel.MapType(cel.StringType,
				cel.ListType(
					cel.MapType(cel.StringType, cel.AnyType))),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable create env T1: (%w)", err)
	}

	return &evaluator{
		env:   envT1,
		cache: lT1,
	}, nil
}

func (e *evaluator) Evaluate(
	ctx context.Context,
	data map[string][]map[string]any,
	expr string,
) (bool, error) {
	switch expr {
	case AlwaysTrueExpr:
		return true, nil
	case AlwaysFalseExpr:
		return false, nil
	default:
		return e.eval(ctx, expr, map[string]any{
			"report": data,
		})
	}
}

func (e *evaluator) eval(
	ctx context.Context,
	expr string,
	vars map[string]any,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("evaluator eval :%w", err)
	}

	prg, err := e.getProgram(expr)
	if err != nil {
		return false, fmt.Errorf("error while compiling program, invalid expr: (%w)", err)
	}

	out, _, err := prg.ContextEval(ctx, vars)
	if err != nil {
		return false, fmt.Errorf("evaluating error: (%w)", err)
	}

	ans, err := out.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return false, fmt.Errorf("undefined output data: (%w), expected boll value", err)
	}

	//nolint:revive,forcetypeassert // not panic
	return ans.(bool), nil
}

func (e *evaluator) getProgram(
	expr string,
) (cel.Program, error) {
	if prg, ok := e.cache.Get(expr); ok {
		return prg, nil
	}

	ast, iss := e.env.Compile(expr)
	if iss != nil {
		return nil, iss.Err()
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("error while compiling program from ast : %w", err)
	}

	e.cache.Add(expr, prg)

	return prg, nil
}

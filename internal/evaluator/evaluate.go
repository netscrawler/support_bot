package evaluator

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	lru "github.com/hashicorp/golang-lru/v2"
)

// Версия cel.Env для разных типов report
type envVer int

const (
	// t1 = envT1 для data - map[string][]map[string]any
	t1 envVer = 0
	// t2 = envT2 для data - map[string][][]string
	t2 envVer = 1
)

type Evaluator struct {
	envT1   *cel.Env
	envT2   *cel.Env
	cacheT1 *lru.Cache[string, cel.Program]
	cacheT2 *lru.Cache[string, cel.Program]
}

func NewEvaluator() (*Evaluator, error) {
	lT1, err := lru.New[string, cel.Program](15)
	if err != nil {
		return nil, fmt.Errorf("unable create cache: (%w)", err)
	}
	lT2, err := lru.New[string, cel.Program](15)
	if err != nil {
		return nil, fmt.Errorf("unable create cache: (%w)", err)
	}

	envT1, err := cel.NewEnv(
		cel.StdLib(),
		ext.Lists(),
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
	envT2, err := cel.NewEnv(
		cel.StdLib(),
		ext.Lists(),
		cel.Macros(cel.StandardMacros...),
		cel.Variable(
			"report",
			cel.MapType(cel.StringType,
				cel.ListType(cel.ListType(cel.AnyType))),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable create env T2: (%w)", err)
	}

	return &Evaluator{envT1: envT1, envT2: envT2, cacheT1: lT1, cacheT2: lT2}, nil
}

func (e *Evaluator) Evaluate(
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
		return e.eval(ctx, t1, expr, map[string]any{
			"report": data,
		})

	}
}

func (e *Evaluator) EvaluateMatrix(
	ctx context.Context,
	data map[string][][]string,
	expr string,
) (bool, error) {
	switch expr {
	case AlwaysTrueExpr:
		return true, nil
	case AlwaysFalseExpr:
		return false, nil
	default:
		return e.eval(ctx, t2, expr, map[string]any{
			"report": data,
		})
	}
}

func (e *Evaluator) eval(
	ctx context.Context,
	envVer envVer,
	expr string,
	vars map[string]any,
) (bool, error) {
	prg, err := e.getProgram(envVer, expr)
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

	return ans.(bool), nil
}

func (e *Evaluator) getProgram(
	envVer envVer,
	expr string,
) (cel.Program, error) {
	var env *cel.Env
	var cache *lru.Cache[string, cel.Program]

	switch envVer {
	case t1:
		env = e.envT1
		cache = e.cacheT1
	case t2:
		env = e.envT2
		cache = e.cacheT2
	default:
		return nil, errors.New("unknown env")
	}

	if prg, ok := cache.Get(expr); ok {
		return prg, nil
	}

	ast, iss := env.Compile(expr)
	if iss != nil {
		return nil, iss.Err()
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	cache.Add(expr, prg)
	return prg, nil
}

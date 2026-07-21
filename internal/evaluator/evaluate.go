package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/ext"
	lru "github.com/hashicorp/golang-lru/v2"
)

type Engine struct {
	env   *cel.Env
	cache *lru.Cache[string, cel.Program]
}

func NewEngine() (*Engine, error) {
	lT1, err := lru.New[string, cel.Program](3)
	if err != nil {
		return nil, fmt.Errorf("unable create cache: (%w)", err)
	}

	opts := []cel.EnvOption{
		cel.StdLib(),
		ext.Lists(),
		ext.Sets(),
		ext.TwoVarComprehensions(),
		cel.OptionalTypes(),
		cel.Macros(cel.StandardMacros...),
		cel.Variable(
			"report",
			cel.MapType(
				cel.StringType,
				cel.ListType(
					cel.MapType(cel.StringType, cel.AnyType),
				),
			),
		),
	}

	opts = append(opts, Builtins()...)

	envT1, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, fmt.Errorf("unable create env: (%w)", err)
	}

	return &Engine{
		env:   envT1,
		cache: lT1,
	}, nil
}

func (e *Engine) EvalStr(ctx context.Context, expression string) (string, error) {
	if strings.HasPrefix(expression, "=") {
		expression = expression[1:]
	} else {
		return expression, nil
	}
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return expression, issues.Err()
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return expression, err
	}

	out, _, err := prg.ContextEval(ctx, map[string]any{})
	if err != nil {
		return expression, err
	}

	return ResultString(out)
}

func (e *Engine) Evaluate(
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

func (e *Engine) eval(
	ctx context.Context,
	expr string,
	vars map[string]any,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("Engine eval :%w", err)
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

func (e *Engine) getProgram(
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

func ResultString(v ref.Val) (string, error) {
	if v == nil {
		return "", nil
	}

	switch x := v.(type) {
	case types.String:
		return string(x), nil

	case types.Bytes:
		return string(x), nil

	case types.Int:
		return strconv.FormatInt(int64(x), 10), nil

	case types.Uint:
		return strconv.FormatUint(uint64(x), 10), nil

	case types.Double:
		return strconv.FormatFloat(float64(x), 'f', -1, 64), nil

	case types.Bool:
		return strconv.FormatBool(bool(x)), nil

	case types.Timestamp:
		return x.Time.Format(time.RFC3339Nano), nil

	case types.Duration:
		return time.Duration(x.Duration).String(), nil

	case types.Null:
		return "", nil

	case traits.Lister:
		native, _ := v.ConvertToNative(reflect.TypeOf([]any{}))
		b, _ := json.Marshal(native)
		return string(b), nil

	case traits.Mapper:
		native, _ := v.ConvertToNative(reflect.TypeOf(map[string]any{}))
		b, _ := json.Marshal(native)
		return string(b), nil
	}

	if s, ok := v.(fmt.Stringer); ok {
		return s.String(), nil
	}

	if native, err := v.ConvertToNative(reflect.TypeOf("")); err == nil {
		return native.(string), nil
	}

	if native, err := v.ConvertToNative(reflect.TypeOf([]byte{})); err == nil {
		return string(native.([]byte)), nil
	}

	if native, err := v.ConvertToNative(reflect.TypeOf((*any)(nil)).Elem()); err == nil {
		b, err := json.Marshal(native)
		if err == nil {
			return string(b), nil
		}
	}

	return fmt.Sprint(v.Value()), nil
}

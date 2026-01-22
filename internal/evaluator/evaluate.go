package evaluator

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	lru "github.com/hashicorp/golang-lru/v2"
)

type Evaluator struct {
	env   *cel.Env
	cache *lru.Cache[string, cel.Program]

	log *slog.Logger
}

func NewEvaluator(log *slog.Logger) (*Evaluator, error) {
	l := log.With(slog.String("module", "evaluator"))

	lT1, err := lru.New[string, cel.Program](15)
	if err != nil {
		l.Error("unable create cache", slog.Any("error", err))

		return nil, fmt.Errorf("unable create cache: (%w)", err)
	}

	l.Info("start creating env")

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
		l.Error("error while creating env 1", slog.Any("error", err))

		return nil, fmt.Errorf("unable create env T1: (%w)", err)
	}

	l.Info("evaluator created")

	return &Evaluator{
		env:   envT1,
		cache: lT1,
		log:   l,
	}, nil
}

func (e *Evaluator) Evaluate(
	ctx context.Context,
	data map[string][]map[string]any,
	expr string,
) (bool, error) {
	e.log.InfoContext(ctx, "evaluating expr", slog.Any("expr", expr))

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

func (e *Evaluator) eval(
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

	out, details, err := prg.ContextEval(ctx, vars)
	if err != nil {
		e.log.ErrorContext(
			ctx,
			"error while eval program",
			slog.Any("expr", expr),
			slog.Any("error", err),
			slog.Any("details", details),
		)

		return false, fmt.Errorf("evaluating error: (%w)", err)
	}

	e.log.InfoContext(
		ctx,
		"program eval success",
		slog.Any("expr", expr),
		slog.Any("details", details),
		slog.Any("out", out),
	)

	ans, err := out.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		e.log.ErrorContext(
			ctx,
			"error while converting out to native bool value",
			slog.Any("error", err),
		)

		return false, fmt.Errorf("undefined output data: (%w), expected boll value", err)
	}

	return ans.(bool), nil
}

func (e *Evaluator) getProgram(
	expr string,
) (cel.Program, error) {
	if prg, ok := e.cache.Get(expr); ok {
		e.log.Debug("program get from cache")

		return prg, nil
	}

	e.log.Debug("cache miss, try compile program from expr")

	ast, iss := e.env.Compile(expr)
	if iss != nil {
		e.log.Error("error while compiling program to ast", slog.Any("error", iss))

		return nil, iss.Err()
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		e.log.Error("error while compiling program from ast", slog.Any("error", err))

		return nil, err
	}

	e.log.Debug("adding program to cache", slog.Any("expr", expr))
	e.cache.Add(expr, prg)

	return prg, nil
}

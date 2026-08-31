package evaluator

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
)

// RowEngine evaluates boolean conditional-format rules against one table row at
// a time. Unlike Engine, whose single variable is the whole report, its
// environment binds three:
//
//	row    map(string, dyn)         the current row's fields
//	value  dyn                      the current cell's raw value (null for a row rule)
//	rows   list(map(string, dyn))   every row in the table block
//
// Expressions are compiled once through Compile and cached; Match then runs the
// compiled program per row. cel-go returns an error (not false) for a missing
// map key, so an author guards with has(row.field). An "is the maximum" style
// check has no max() built-in here — use a comprehension, e.g.
// rows.all(r, r.accepted <= value).
type RowEngine struct {
	env *cel.Env

	mu    sync.RWMutex
	progs map[string]cel.Program
}

// NewRowEngine builds the row-scoped CEL environment. It mirrors the extension
// set of NewEngine (lists, sets, two-variable comprehensions, optionals, the
// project builtins) so expressions read the same across the two engines.
func NewRowEngine() (*RowEngine, error) {
	rowType := cel.MapType(cel.StringType, cel.DynType)

	opts := []cel.EnvOption{
		cel.StdLib(),
		ext.Lists(),
		ext.Sets(),
		ext.TwoVarComprehensions(),
		cel.OptionalTypes(),
		cel.Macros(cel.StandardMacros...),
		cel.Variable("row", rowType),
		cel.Variable("value", cel.DynType),
		cel.Variable("rows", cel.ListType(rowType)),
	}
	opts = append(opts, Builtins()...)

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, fmt.Errorf("row engine: new env: %w", err)
	}

	return &RowEngine{env: env, progs: make(map[string]cel.Program)}, nil
}

// Compile parses and caches every expression up front so a syntax or type error
// surfaces before any row is evaluated.
func (e *RowEngine) Compile(exprs ...string) error {
	for _, expr := range exprs {
		if _, err := e.program(expr); err != nil {
			return err
		}
	}

	return nil
}

// Match reports whether expr is true for the given row. value is the current
// cell (nil for a row-scoped rule); rows is the block's full row set. A missing
// key, a non-boolean result, or any other evaluation failure is returned as an
// error with the bool false — the caller treats that as "no match".
func (e *RowEngine) Match(
	ctx context.Context,
	expr string,
	row map[string]any,
	value any,
	rows []map[string]any,
) (bool, error) {
	prg, err := e.program(expr)
	if err != nil {
		return false, err
	}

	out, _, err := prg.ContextEval(ctx, map[string]any{
		"row":   row,
		"value": value,
		"rows":  rows,
	})
	if err != nil {
		return false, fmt.Errorf("row engine: eval %q: %w", expr, err)
	}

	native, err := out.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return false, fmt.Errorf("row engine: %q did not yield a bool: %w", expr, err)
	}

	b, ok := native.(bool)
	if !ok {
		return false, fmt.Errorf("row engine: %q did not yield a bool", expr)
	}

	return b, nil
}

func (e *RowEngine) program(expr string) (cel.Program, error) {
	e.mu.RLock()
	prg, ok := e.progs[expr]
	e.mu.RUnlock()

	if ok {
		return prg, nil
	}

	ast, iss := e.env.Compile(expr)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("row engine: compile %q: %w", expr, iss.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("row engine: program %q: %w", expr, err)
	}

	e.mu.Lock()
	e.progs[expr] = prg
	e.mu.Unlock()

	return prg, nil
}

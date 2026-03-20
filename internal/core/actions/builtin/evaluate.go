package builtin

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	lru "github.com/hashicorp/golang-lru/v2"
)

// Evaluator предоставляет функциональность для оценки необходимости отправки отчета
// на основе CEL (Common Expression Language) выражений.
//
// Основная задача модуля - анализ структурированных данных отчета и принятие решения
// о необходимости его отправки на основе заданных правил.
//
// Входные данные:
//
// На вход получает структуру map[string][]map[string]any, где:
//   - Ключ первого уровня - название листа(запроса в бд queries поле - title)
//   - Значение - массив записей (каждая запись это map[string]any)
//
// Пример:
//
//	{
//	  "sheet1": {},
//	  "sheet2": {
//	    {"total": 0, "count": 5},
//	    {"total": 10, "count": 0},
//	  },
//	}
//
// CEL окружение:
//
// Работает через [go-cel](https://github.com/google/cel-go) библиотеку с подключенными модулями:
//   - cel.StdLib() - стандартная библиотека CEL
//   - ext.Lists() - расширенные операции со списками
//   - ext.Sets() - операции с множествами
//   - ext.TwoVarComprehensions() - двухпараметровые comprehensions
//   - cel.OptionalTypes() - поддержка опциональных типов
//   - cel.Macros(cel.StandardMacros...) - стандартные макросы
//
// Доступная переменная в выражениях:
//   - report - map[string][]map[string]any с данными отчета
//
// Специальные выражения:
//   - "[*]" (AlwaysTrueExpr) - всегда возвращает true
//   - "[!*]" (AlwaysFalseExpr) - всегда возвращает false
//
// Примеры CEL выражений:
//
//	1)
//
//	report := map[string][]map[string]any{
//		"sheet1": {
//			{"total": 0, "count": 5},
//			{"total": 10, "count": 0},
//		},
//	}
//
//	// Проверка что все записи в sheet1 имеют total != 0
//	report["sheet1"].all(r, r["total"] != 0)
//
//	2)
//
//	report := map[string][]map[string]any{
//		"sheet1": {},
//	}
//
//	// Проверка количества записей в конкретном листе
//	size(report["sheet1"]) > 1 -> Вернет false
//
//	3)
//
//	report := map[string][]map[string]any{
//		"sheet1": {},
//		"sheet2": {
//			{"total": 0, "count": 5},
//			{"total": 10, "count": 0},
//		},
//	}
//
// Подсчет общего количества записей во всех листах
//
//	report.map(k, report[k]).flatten().size() > 1 -> Вернет true
//
// Кеширование:
//
// # Модуль использует LRU кеш (размер 15) для хранения скомпилированных CEL программ
//
// Использование:
//
//	log := slog.Default()
//	eval, err := evaluator.NewEvaluator(log)
//	if err != nil {
//	    // обработка ошибки
//	}
//
//	report := map[string][]map[string]any{
//	    "sheet1": {
//	        {"total": 10, "count": 5},
//	    },
//	}
//
//	expr := `size(report["sheet1"]) > 0`
//	shouldSend, err := eval.Evaluate(ctx, report, expr)
//	if err != nil {}

const (
	// Returning always true result of eval.
	AlwaysTrueExpr = "[*]"
	// Returning always false result of eval.
	AlwaysFalseExpr = "[!*]"
)

type EvaluationModule struct {
	env   *cel.Env
	cache *lru.Cache[string, cel.Program]

	log *slog.Logger
}

func NewEvaluator(log *slog.Logger) (*EvaluationModule, error) {
	l := log.With(slog.String("module", "evaluator"))

	lT1, err := lru.New[string, cel.Program](3)
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

	return &EvaluationModule{
		env:   envT1,
		cache: lT1,
		log:   l,
	}, nil
}

func (e *EvaluationModule) Evaluate(
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

func (e *EvaluationModule) eval(
	ctx context.Context,
	expr string,
	vars map[string]any,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("evaluator eval :%w", err)
	}

	prg, err := e.getProgram(ctx, expr)
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

	//nolint:revive,forcetypeassert // not panic
	return ans.(bool), nil
}

func (e *EvaluationModule) getProgram(
	ctx context.Context,
	expr string,
) (cel.Program, error) {
	if prg, ok := e.cache.Get(expr); ok {
		e.log.DebugContext(ctx, "program get from cache")

		return prg, nil
	}

	e.log.DebugContext(ctx, "cache miss, try compile program from expr")

	ast, iss := e.env.Compile(expr)
	if iss != nil {
		e.log.ErrorContext(ctx, "error while compiling program to ast", slog.Any("error", iss))

		return nil, iss.Err()
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		e.log.ErrorContext(ctx, "error while compiling program from ast", slog.Any("error", err))

		return nil, err
	}

	e.log.DebugContext(ctx, "adding program to cache", slog.Any("expr", expr))
	e.cache.Add(expr, prg)

	return prg, nil
}

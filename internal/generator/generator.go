package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"support_bot/internal/collector"
	"support_bot/internal/exporter"
	"support_bot/internal/models"
	"support_bot/internal/processor"
	"time"
)

// generationTimeout ограничивает время генерации одного отчёта: без него
// зависшая генерация (SQL/Lua-пайплайн, CEL, экспорт) навсегда занимает
// слот воркера в общем пуле (см. orchestrator.go).
const generationTimeout = 5 * time.Minute

type Collector interface {
	Collect(ctx context.Context, cards ...models.Card) (models.Dataset, error)
}

type Evaluator interface {
	Evaluate(
		ctx context.Context,
		data models.Dataset,
		expr string,
	) (bool, error)
	EvalStr(ctx context.Context, expr string) (string, error)
}

// job — единица работы, отправляемая в общий пул воркеров Generator. result
// получает собранный dataset, экспортированные файлы и результат оценки —
// доставка получателям остается ответственностью вызывающего кода. Каждый
// вызывающий (internal/orchestrator, и для событийного, и для разового
// пути) идет через один и тот же метод Generate ниже, поэтому в этом API
// нет понятия "типа" генерации.
type job struct {
	ctx    context.Context
	report models.Report
	result chan<- jobResult
}

type jobResult struct {
	dataset models.Dataset
	data    []models.Data
	approve bool
	err     error
}

type Generator struct {
	c chan job

	clct Collector

	eval Evaluator

	proc *processor.Processor

	numWorkers uint8

	log *slog.Logger
}

func New(
	clct Collector,
	proc *processor.Processor,
	eval Evaluator,
	workers uint8,
	log *slog.Logger,
) *Generator {
	l := log.With(slog.Any("module", "generator"))

	if workers == 0 {
		workers = 1
	}

	return &Generator{
		c:          make(chan job),
		clct:       clct,
		eval:       eval,
		log:        l,
		numWorkers: workers,
		proc:       proc,
	}
}

func (g *Generator) Start(ctx context.Context) {
	for i := range g.numWorkers {
		go g.worker(ctx, g.c, i)
	}
}

// Generate отправляет report в общий пул воркеров и блокируется до его
// обработки, возвращая собранный dataset, экспортированные файлы и
// результат оценки условия отправки. Метод не делает никакой
// специфичной для вызывающего кода интерпретации результата — доставка
// отчета и любая семантика "не найдено" остаются ответственностью
// вызывающего кода.
func (g *Generator) Generate(
	ctx context.Context,
	report models.Report,
) (models.Dataset, []models.Data, bool, error) {
	result := make(chan jobResult, 1)

	select {
	case g.c <- job{ctx: ctx, report: report, result: result}:
	case <-ctx.Done():
		return nil, nil, false, ctx.Err()
	}

	select {
	case r := <-result:
		return r.dataset, r.data, r.approve, r.err
	case <-ctx.Done():
		return nil, nil, false, ctx.Err()
	}
}

// generate выполняет общий пайплайн отчета: разрешает параметры запросов,
// собирает данные, прогоняет их через pipeline обработки (если он задан),
// оценивает условие отчета и экспортирует результат. Функция намеренно не
// занимается доставкой — доставка получателям и любая семантика "не
// найдено" остаются ответственностью вызывающего кода (см.
// internal/orchestrator, который обрабатывает и событийный, и разовый
// пути).
func (g *Generator) generate(
	ctx context.Context,
	report models.Report,
) (data models.Dataset, res []models.Data, approve bool, err error) {
	l := g.log
	l.DebugContext(ctx, "start generating report", slog.Any("report", report))

	var queries []models.Card

	for _, q := range report.Queries {
		err := q.ResolveParams(ctx, g.eval)
		if err != nil {
			l.ErrorContext(
				ctx,
				"resolving query params",
				slog.Any("error", err),
				slog.Any("query", q),
			)
		}

		queries = append(queries, q)
	}

	data, err = g.clct.Collect(ctx, queries...)
	if err != nil && !errors.Is(err, collector.ErrEmtyCard) {
		l.ErrorContext(ctx, "error while collect data", slog.Any("error", err))

		return nil, nil, false, err
	}

	if report.Pipeline != nil {
		l.InfoContext(
			ctx,
			"start pipeline for report",
			slog.Any("pipeline", report.Pipeline.Name),
			slog.Any("report", report.Name),
		)

		processed, err := g.proc.Process(ctx, data, report.Pipeline)
		if err != nil {
			l.ErrorContext(
				ctx,
				"pipeline execution error, stop generate report",
				slog.Any("error", err),
			)

			return nil, nil, false, err
		}
		data = processed
	}

	approve, err = g.eval.Evaluate(ctx, data, report.Evaluation)
	if err != nil {
		l.ErrorContext(ctx, "error while evaluate report", slog.Any("error", err))

		return nil, nil, false, err
	}

	if !approve {
		l.InfoContext(ctx, "negative result of evaluating, don`t send report")

		return data, nil, false, nil
	}

	res = make([]models.Data, 0, len(report.Exports))

	for _, e := range report.Exports {
		r, err := exporter.Export(data, e)
		if err != nil {
			l.ErrorContext(
				ctx,
				"error while export report",
				slog.Any("error", err),
				slog.Any("export", e),
			)

			continue
		}

		res = append(res, r...)
	}

	return data, res, true, nil
}

func (g *Generator) worker(ctx context.Context, jobs <-chan job, id uint8) {
	g.log.DebugContext(ctx, fmt.Sprintf("start worker %d", id))

	for {
		select {
		case <-ctx.Done():
			g.log.DebugContext(ctx, "context cancelled")

			return
		case j, ok := <-jobs:
			if !ok {
				g.log.DebugContext(ctx, "jobs chan closed")

				return
			}

			rCtx, cancel := context.WithTimeout(j.ctx, generationTimeout)
			dataset, res, approve, err := g.generate(rCtx, j.report)
			cancel()

			j.result <- jobResult{dataset: dataset, data: res, approve: approve, err: err}
		}
	}
}

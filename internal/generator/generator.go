package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"support_bot/internal/collector"
	"support_bot/internal/exporter"
	"support_bot/internal/models"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/processor"
	"time"
)

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

type Generator struct {
	c chan Job

	clct Collector

	eval Evaluator

	proc *processor.Processor

	numWorkers uint8

	log *slog.Logger
}

func New(
	c chan Job,
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
		c:          c,
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

func (g *Generator) worker(ctx context.Context, jobs <-chan Job, id uint8) {
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

			rCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			rvCtx := logger.AppendCtx(rCtx, slog.Any("report_name", j.Report.Name))

			dataset, res, approve, err := g.generate(rvCtx, j.Report)
			j.Result <- JobResult{Dataset: dataset, Data: res, Approve: approve, Err: err}

			cancel()
		}
	}
}

// generate runs the shared report pipeline: resolve query params, collect
// data, run the processing pipeline (if any), evaluate the report
// condition, and export the result. It stops short of delivering anything —
// delivery to recipients is the caller's responsibility (see
// internal/orchestrator for the scheduled path, and GenerateOnDemand in
// on_demand.go for the on-demand path).
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

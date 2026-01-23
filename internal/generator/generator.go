package generator

import (
	"context"
	"fmt"
	"log/slog"
	"support_bot/internal/delivery"
	"support_bot/internal/exporter"
	"time"

	models "support_bot/internal/models/report"
)

type Sender interface {
	Send(
		ctx context.Context,
		meta []models.Targeted,
		data []models.ReportData,
	) error
}

type Collector interface {
	Collect(ctx context.Context, cards ...models.Card) (map[string][]map[string]any, error)
}

type Evaluator interface {
	Evaluate(
		ctx context.Context,
		data map[string][]map[string]any,
		expr string,
	) (bool, error)
}

type Generator struct {
	c chan models.Report

	clct Collector

	eval Evaluator

	snd Sender

	numWorkers uint8

	log *slog.Logger
}

func New(
	c chan models.Report,
	clct Collector,
	snd Sender,
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
		snd:        snd,
		log:        l,
		numWorkers: workers,
	}
}

func (g *Generator) Start(ctx context.Context) {
	for i := range g.numWorkers {
		go g.worker(ctx, g.c, i)
	}
}

func (g *Generator) worker(ctx context.Context, jobs <-chan models.Report, id uint8) {
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

			err := g.createReport(rCtx, j)
			if err != nil {
				g.log.ErrorContext(ctx, "error create report", slog.Any("error", err))
			}

			cancel()
		}
	}
}

func (g *Generator) createReport(ctx context.Context, report models.Report) error {
	l := g.log.With("report", report.Name)
	l.DebugContext(ctx, "start generating report", slog.Any("report", report))

	data, err := g.clct.Collect(ctx, report.Queries...)
	if err != nil {
		l.ErrorContext(ctx, "error while collect data", slog.Any("error", err))

		return err
	}

	approve, err := g.eval.Evaluate(ctx, data, report.Evaluation)
	if err != nil {
		l.ErrorContext(ctx, "error while evaluate report", slog.Any("error", err))

		return err
	}

	if !approve {
		l.InfoContext(ctx, "negative result of evaluating, don`t send report")

		return nil
	}

	res := make([]models.ReportData, 0, len(report.Exports))

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

		res = append(res, r)
	}

	targets, err := delivery.GetTarget(report.Recipients...)
	if err != nil {
		l.ErrorContext(ctx, "error while resolving targets", slog.Any("error", err))
	}

	if len(targets) == 0 {
		l.ErrorContext(ctx, "emty targets list")

		return fmt.Errorf("emty targets list")
	}

	l.DebugContext(ctx, "sending report", slog.Any("targets", targets), slog.Any("report", res))

	err = g.snd.Send(ctx, targets, res)
	if err != nil {
		l.ErrorContext(ctx, "error while send report", slog.Any("error", err))

		return err
	}

	l.InfoContext(ctx, "report generated")

	return nil
}

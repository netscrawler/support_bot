package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"support_bot/internal/collector"
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

type Exporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

type Generator struct {
	c chan models.Report

	clct Collector

	eval Evaluator

	snd models.SenderProvider

	exporter Exporter

	proc *processor.Processor

	numWorkers uint8

	sentMsgRepo SentMsgRepository

	log *slog.Logger
}

func New(
	c chan models.Report,
	clct Collector,
	exp Exporter,
	snd models.SenderProvider,
	sendRepo SentMsgRepository,
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
		c:           c,
		clct:        clct,
		eval:        eval,
		snd:         snd,
		exporter:    exp,
		log:         l,
		numWorkers:  workers,
		sentMsgRepo: sendRepo,
		proc:        proc,
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
			rvCtx := logger.AppendCtx(rCtx, slog.Any("report_name", j.Name))

			err := g.createReport(rvCtx, j)
			if err != nil {
				g.log.ErrorContext(rvCtx, "error create report", slog.Any("error", err))
			}

			cancel()
		}
	}
}

func (g *Generator) createReport(ctx context.Context, report models.Report) error {
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

	data, err := g.clct.Collect(ctx, queries...)
	if err != nil && !errors.Is(err, collector.ErrEmtyCard) {
		l.ErrorContext(ctx, "error while collect data", slog.Any("error", err))

		return err
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

			return err
		}
		data = processed
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

	res := make([]models.Data, 0, len(report.Exports))

	for _, e := range report.Exports {
		r, err := g.exporter.Export(data, e)
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

	if len(report.Recipients) == 0 {
		l.ErrorContext(ctx, "empty targets list")

		return fmt.Errorf("empty targets list")
	}

	// Достаем "_meta" лист из данных, для использования в шаблоне email
	addMeta := make(map[string]any)
	meta, ok := data["_meta"]
	if ok {
		for _, d := range meta {
			maps.Insert(addMeta, maps.All(d))
		}
	}
	l.InfoContext(ctx, "meta", slog.Any("meta", addMeta), slog.Any("_meta", data["_meta"]))
	msg := models.NewMessage(report.Name, res, addMeta, report.Recipients...)

	resMsg, err := msg.Send(ctx, g.snd)
	if err != nil {
		l.ErrorContext(ctx, "error while send message", slog.Any("error", err))
	}

	if len(resMsg) == 0 {
		l.InfoContext(ctx, "report generated")

		return nil
	}

	l.InfoContext(
		ctx,
		"saving message to database",
		slog.Any("report", report.Name),
		slog.Any("message", resMsg),
	)

	err = g.sentMsgRepo.saveTgMsg(ctx, msg.ReportName, resMsg)
	if err != nil {
		l.WarnContext(ctx, "result msg save failed", slog.Any("error", err))
	}

	return nil
}

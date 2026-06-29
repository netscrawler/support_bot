package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"support_bot/internal/collector"
	"support_bot/internal/exporter"
	"support_bot/internal/models"
	"support_bot/internal/pkg/logger"
)

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

	snd models.SenderProvider

	numWorkers uint8

	sentMsgRepo SentMsgRepository

	log *slog.Logger
}

func New(
	c chan models.Report,
	clct Collector,
	snd models.SenderProvider,
	sendRepo SentMsgRepository,
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
		log:         l,
		numWorkers:  workers,
		sentMsgRepo: sendRepo,
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

	data, err := g.clct.Collect(ctx, report.Queries...)
	if err != nil && !errors.Is(err, collector.ErrEmtyCard) {
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

	res := make([]models.Data, 0, len(report.Exports))

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

	if len(report.Recipients) == 0 {
		l.ErrorContext(ctx, "empty targets list")

		return fmt.Errorf("empty targets list")
	}

	msg := models.NewMessage(report.Name, res, report.Recipients...)

	resMsg, err := msg.Send(ctx, g.snd)
	if err != nil {
		l.ErrorContext(ctx, "error while send message", slog.Any("error", err))
	}

	if len(resMsg) == 0 {
		l.InfoContext(ctx, "report generated")

		return nil
	}

	err = g.sentMsgRepo.saveTgMsg(ctx, msg.ReportName, resMsg)
	if err != nil {
		l.WarnContext(ctx, "result msg save failed", slog.Any("error", err))
	}

	return nil
}

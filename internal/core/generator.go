package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"support_bot/internal/core/workflow"
	"support_bot/internal/delivery"
	"support_bot/internal/pkg/logger"
	plugins "support_bot/internal/plugin"
	"time"

	models "support_bot/internal/models/report"

	"github.com/google/uuid"
)

type Sender interface {
	Send(
		ctx context.Context,
		meta []models.Targeted,
		data any,
	) error
}

type Generator struct {
	c chan models.Report

	engine *workflow.Engine

	pManager plugins.Manager

	snd Sender

	numWorkers uint8

	log *slog.Logger
}

func New(
	c chan models.Report,
	snd Sender,
	workers uint8,
	log *slog.Logger,
) *Generator {
	l := log.With(slog.Any("module", "generator"))

	if workers == 0 {
		workers = 1
	}

	return &Generator{
		c:          c,
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
			rvCtx := logger.AppendCtx(rCtx, slog.String("report_name", j.Name), slog.String("report_trace_id", uuid.NewString()))

			err := g.createReport(rvCtx, j)
			if err != nil {
				g.log.ErrorContext(rvCtx, "error create report", slog.Any("error", err))
			}

			cancel()
		}
	}
}

func (g *Generator) createReport(ctx context.Context, report models.Report) error {
	g.log.DebugContext(ctx, "start generating report", slog.Any("report", report))

	res, err := g.engine.Run(ctx, report.Workflow, report.Meta)
	if err != nil {
		if !errors.Is(err, workflow.ErrEndOutputNotFound) {
			return err
		}

		g.log.ErrorContext(ctx, "workflow output not found", slog.Any("error", err))

		if err := g.saveResult(ctx, res); err != nil {
			g.log.ErrorContext(ctx, "saving history failed", slog.Any("error", err))
		}
	}

	targets, err := delivery.GetTarget(report.Recipients...)
	if err != nil {
		return fmt.Errorf("error while resolving targets: %w", err)
	}

	if len(targets) == 0 {
		return fmt.Errorf("emty targets list")
	}

	g.log.DebugContext(ctx, "sending report", slog.Any("targets", targets), slog.Any("report", res))

	err = g.snd.Send(ctx, targets, res.Result)
	if err != nil {
		g.log.ErrorContext(ctx, "error while send report", slog.Any("error", err))

		return err
	}

	g.log.InfoContext(ctx, "report generated")

	return nil
}

// TODO: implement
func (g *Generator) saveResult(ctx context.Context, history *workflow.RunHistory) error {
	return fmt.Errorf("not implemented")
}

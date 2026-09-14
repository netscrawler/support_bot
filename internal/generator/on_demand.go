package generator

import (
	"context"
	"support_bot/internal/models"
)

// Job is a unit of work sent through Generator's shared worker pool. Result
// is nil for fire-and-forget scheduled reports (the orchestrator's normal
// flow); when set, the worker writes the outcome there instead of
// delivering to recipients, so on-demand callers share the exact same
// concurrency-limited pool — and the same 5-minute per-job timeout — as
// scheduled generation.
type Job struct {
	Report models.Report
	Result chan<- JobResult
}

type JobResult struct {
	Data []models.Data
	Err  error
}

// generateOnDemand runs the shared report pipeline and returns its exported
// files without delivering them to the report's recipients. A negative
// evaluation result (nothing to report) is surfaced as models.ErrNotFound,
// matching the "not found" semantics the HTTP layer expects.
func (g *Generator) generateOnDemand(ctx context.Context, report models.Report) ([]models.Data, error) {
	_, res, approve, err := g.generate(ctx, report)
	if err != nil {
		return nil, err
	}

	if !approve || len(res) == 0 {
		return nil, models.ErrNotFound
	}

	return res, nil
}

// GenerateOnDemand enqueues report onto the same channel the scheduled
// worker pool consumes and blocks for the result, so an on-demand request
// is bound by the same numWorkers concurrency limit as scheduled
// generation, and queues behind/ahead of scheduled jobs fairly.
func (g *Generator) GenerateOnDemand(ctx context.Context, report models.Report) ([]models.Data, error) {
	result := make(chan JobResult, 1)

	select {
	case g.c <- Job{Report: report, Result: result}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case r := <-result:
		return r.Data, r.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReportGeneratorAdapter adapts Generator to service.ReportGenerator's
// single-file Generate signature (internal/service/report_generator.go).
// On-demand reports are expected to declare exactly one export; if more are
// configured, the first is returned — a known simplification.
type ReportGeneratorAdapter struct {
	Gen *Generator
}

func (a ReportGeneratorAdapter) Generate(ctx context.Context, report models.Report) (models.Data, error) {
	res, err := a.Gen.GenerateOnDemand(ctx, report)
	if err != nil {
		return models.Data{}, err
	}

	if len(res) == 0 {
		return models.Data{}, models.ErrNotFound
	}

	return res[0], nil
}

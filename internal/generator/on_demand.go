package generator

import (
	"context"
	"support_bot/internal/models"
)

// Job is a unit of work sent through Generator's shared worker pool. Result
// receives the generated dataset, exported files, and evaluation outcome —
// delivering them to recipients is the caller's responsibility (see
// internal/orchestrator for the scheduled path), so both that path and
// on-demand callers share the exact same concurrency-limited pool — and the
// same 5-minute per-job timeout.
type Job struct {
	Report models.Report
	Result chan<- JobResult
}

type JobResult struct {
	Dataset models.Dataset
	Data    []models.Data
	Approve bool
	Err     error
}

// GenerateOnDemand enqueues report onto the same channel the scheduled
// worker pool consumes and blocks for the result, so an on-demand request
// is bound by the same numWorkers concurrency limit as scheduled
// generation, and queues behind/ahead of scheduled jobs fairly. A negative
// evaluation result (nothing to report) is surfaced as models.ErrNotFound,
// matching the "not found" semantics the HTTP layer expects.
func (g *Generator) GenerateOnDemand(ctx context.Context, report models.Report) ([]models.Data, error) {
	result := make(chan JobResult, 1)

	select {
	case g.c <- Job{Report: report, Result: result}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case r := <-result:
		if r.Err != nil {
			return nil, r.Err
		}

		if !r.Approve || len(r.Data) == 0 {
			return nil, models.ErrNotFound
		}

		return r.Data, nil
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

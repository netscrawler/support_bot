package processor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"support_bot/internal/models"
	"support_bot/internal/processor/pipeline"
)

type RunnerRegistry struct {
	reg map[string]pipeline.Runner
}

func NewReg() *RunnerRegistry {
	return &RunnerRegistry{
		reg: make(map[string]pipeline.Runner),
	}
}

func (r *RunnerRegistry) Register(name string, runner pipeline.Runner) {
	r.reg[strings.ToLower(name)] = runner
}

func (r *RunnerRegistry) Get(name string) (pipeline.Runner, error) {
	rn, ok := r.reg[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("runner not found: %s", name)
	}

	return rn, nil
}

type Processor struct {
	reg *RunnerRegistry
	log *slog.Logger
}

func NewProcessor(reg *RunnerRegistry, log *slog.Logger) *Processor {
	return &Processor{
		reg: reg,
		log: log,
	}
}

func (p *Processor) Process(
	ctx context.Context,
	data models.Dataset,
	pipeline *pipeline.Pipeline,
) (models.Dataset, error) {
	p.log.InfoContext(
		ctx,
		"start pipeline",
		slog.Any("pipeline_name", pipeline.Name),
		slog.Any("steps_count", len(pipeline.Steps)),
	)

	defer p.log.InfoContext(
		ctx,
		"finish pipeline",
		slog.Any("pipeline_name", pipeline.Name),
	)

	for _, step := range pipeline.Steps {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("stopped on step %s: %w", step.ID, err)
		}
		rn, err := p.reg.Get(step.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get runner for step %s: %w", step.Type, err)
		}
		out, err := rn.Run(ctx, step, data)
		if err != nil {
			return nil, fmt.Errorf("failed pipeline on step %s: %w", step.ID, err)
		}
		data = out
	}

	return data, nil
}

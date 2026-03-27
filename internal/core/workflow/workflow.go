package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"support_bot/internal/core/actions"
	"support_bot/internal/core/workflow/definition"
	"support_bot/internal/core/workflow/execution"
	"support_bot/internal/core/workflow/executor"
	"support_bot/internal/core/workflow/graph"
	"support_bot/internal/core/workflow/registry"
	"support_bot/internal/core/workflow/scheduler"
	"support_bot/internal/core/workflow/validator"
)

type externalActionProvider interface {
	GetAction(cxt context.Context, key string) (registry.Action, error)
	RegisterFromDef(ctx context.Context, reg *registry.Registry, def *definition.WorkflowDef) error
}

var ErrEndOutputNotFound = errors.New("end output not found")

// Engine is the top-level workflow engine.
//
// Create one instance per application (it is stateless between runs) and share
// it. Register built-in and plugin actions into the Registry before the first
// call to Run.
//
//	reg := registry.New()
//	reg.Register("std@collect", actions.NewCollectAction(collector))
//	reg.Register("std@send",    actions.NewSendAction(sender))
//
//	engine := workflow.NewEngine(reg, 8, logger)
//	err := engine.Run(ctx, report.Workflow)
type Engine struct {
	reg     *registry.Registry
	workers int
	log     *slog.Logger

	eP externalActionProvider
}

// RunHistory is the detailed workflow execution result.
type RunHistory struct {
	WorkflowID   string
	ExecutionID  string
	Result       any
	History      []execution.NodeHistoryEntry
	HistoryGraph execution.HistoryGraph
}

// ErrUnknownActionType indicates that a workflow node references an action type
// that is not registered in the engine registry.
var ErrUnknownActionType = errors.New("workflow has unknown action type")

// NewEngine creates a new Engine.
//
//   - reg     — action registry, pre-populated with all supported action types
//   - workers — max parallel nodes (defaults to 4 when <= 0)
//   - log     — structured logger
func NewEngine(reg *registry.Registry, workers int, log *slog.Logger, provider externalActionProvider) *Engine {
	if workers <= 0 {
		workers = 4
	}

	if reg == nil {
		reg = registry.New()
	}

	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &Engine{
		reg:     reg,
		workers: workers,
		log:     log.With(slog.String("module", "workflow.engine")),
		eP:      provider,
	}
}

func (e *Engine) RunSimple(ctx context.Context, raw json.RawMessage, meta ...map[string]string) error {
	_, err := e.RunWithResult(ctx, raw, meta...)

	return err
}

// RunWithResult executes a workflow and returns the final output produced by
// pseudo end node.
func (e *Engine) RunWithResult(ctx context.Context, raw json.RawMessage, meta ...map[string]string) (any, error) {
	runHistory, err := e.Run(ctx, raw, meta...)
	if err != nil {
		return nil, err
	}

	return runHistory.Result, nil
}

// Run executes a workflow described by raw JSON.
//
// The full pipeline:
//  1. Parse   — JSON → WorkflowDef
//  2. Validate — structural integrity (unique IDs, valid edges, no cycles)
//  3. Build   — WorkflowDef → RuntimeGraph (adjacency lists, in-degrees)
//  4. Execute — Scheduler + Executor drive parallel node execution
//
// Returns nil on success. Returns the first node error encountered on failure.
func (e *Engine) Run(ctx context.Context, raw json.RawMessage, meta ...map[string]string) (*RunHistory, error) {
	// 1. parse
	def, err := definition.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("workflow: parse: %w", err)
	}

	// 2. validate
	if err := validator.Validate(def); err != nil {
		return nil, fmt.Errorf("workflow: validate: %w", err)
	}

	if len(meta) > 0 {
		def.AppendMetadata(meta...)
	}

	nReg := e.reg.Clone()
	if err := nReg.RegisterOrReplace(ActionTypeStart, actions.StartAction{}); err != nil {
		return nil, fmt.Errorf("workflow: register pseudo actions: %w", err)
	}

	if err := nReg.RegisterOrReplace(ActionTypeEnd, actions.EndAction{}); err != nil {
		return nil, fmt.Errorf("workflow: register pseudo actions: %w", err)
	}

	if err := e.eP.RegisterFromDef(ctx, nReg, def); err != nil {
		return nil, fmt.Errorf("workflow: register external actions: %w", err)
	}

	if err := validateRegisteredActions(def, nReg); err != nil {
		return nil, fmt.Errorf("workflow: validate: %w", err)
	}

	// 3. build runtime graph
	g := graph.Build(def)

	// 4. create execution instance
	exec := execution.New(def.ID, g)

	e.log.InfoContext(ctx, "workflow started",
		slog.String("workflow_id", def.ID),
		slog.String("execution_id", exec.ID),
		slog.Int("nodes", len(g.Nodes)),
		slog.Int("edges", len(def.Edges)),
		slog.Int("start_nodes", len(g.StartIDs)),
	)

	// 5. create scheduler (buffer == number of nodes — never blocks on enqueue)
	sched := scheduler.New(exec, len(g.Nodes))

	// 6. run executor (blocks until workflow completes or ctx is cancelled)
	ex := executor.New(exec, nReg, sched, e.workers, e.log)

	if err := ex.Run(ctx); err != nil {
		return nil, fmt.Errorf("workflow %q (execution %s): %w", def.ID, exec.ID, err)
	}

	e.log.InfoContext(ctx, "workflow completed",
		slog.String("workflow_id", def.ID),
		slog.String("execution_id", exec.ID),
	)

	result, ok := exec.Context.Get(definition.PseudoEndNodeID)
	if !ok {
		return nil, fmt.Errorf("workflow: %w", ErrEndOutputNotFound)
	}

	return &RunHistory{
		WorkflowID:   def.ID,
		ExecutionID:  exec.ID,
		Result:       result,
		History:      exec.History(),
		HistoryGraph: exec.HistoryGraph(),
	}, nil
}

// Registry returns the underlying action registry.
// Use it to register additional actions after engine creation.
func (e *Engine) Registry() *registry.Registry {
	return e.reg
}

func (e *Engine) collectNeededActionsToRegistry(ctx context.Context, modules []string, reg *registry.Registry) error {
	for _, module := range modules {
		act, err := e.eP.GetAction(ctx, module)
		if err != nil {
			return fmt.Errorf("collect actions to registry: %w", err)
		}

		err = reg.Register(module, act)
		if err != nil {
			return fmt.Errorf("collect actions to registry: %w", err)
		}
	}
	return nil
}

func (e *Engine) getNeededModules(def *definition.WorkflowDef) ([]string, bool) {
	needLoadM := make(map[string]struct{})
	for _, n := range def.Nodes {
		if n.IsPlugin() {
			plug := n.GetTypeName()
			needLoadM[plug] = struct{}{}
		}
	}

	nl := make([]string, 0)
	for k := range maps.Keys(needLoadM) {
		nl = append(nl, k)
	}
	return nl, len(nl) > 0
}

func validateRegisteredActions(def *definition.WorkflowDef, reg *registry.Registry) error {
	for _, node := range def.Nodes {
		if reg.Has(node.Type) {
			continue
		}

		return fmt.Errorf("%w: node %q uses %q", ErrUnknownActionType, node.ID, node.Type)
	}

	return nil
}

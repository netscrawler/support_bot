package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"support_bot/internal/core/actions/builtin"
	"support_bot/internal/core/actions/builtin/exporter"
	"support_bot/internal/core/workflow/registry"
	"support_bot/internal/delivery"

	models "support_bot/internal/models/report"
)

// Collector is the data-collection dependency for std@collect action.
type Collector interface {
	Collect(ctx context.Context, cards ...models.Card) (map[string][]map[string]any, error)
}

// Evaluator is the expression-evaluation dependency for std@evaluate action.
type Evaluator interface {
	Evaluate(ctx context.Context, data map[string][]map[string]any, expr string) (bool, error)
}

// Sender is the delivery dependency for std@send action.
type Sender interface {
	Send(ctx context.Context, metas []models.Targeted, data []models.ExportedReport) error
}

type Saver interface {
	Save(ctx context.Context, id string, data []models.ExportedReport) error
}

// BuiltinDeps groups dependencies required to register standard workflow actions.
type BuiltinDeps struct {
	Collector Collector
	Evaluator Evaluator
	Sender    Sender
	Saver     Saver
}

func RegisterBuiltins(reg *registry.Registry, deps BuiltinDeps) error {
	if reg == nil {
		return errors.New("workflow/actions: registry is nil")
	}

	if deps.Collector == nil {
		return errors.New("workflow/actions: collector is nil")
	}

	if deps.Evaluator == nil {
		return errors.New("workflow/actions: evaluator is nil")
	}

	if deps.Sender == nil {
		return errors.New("workflow/actions: sender is nil")
	}

	if deps.Saver == nil {
		return errors.New("workflow/actions: saver is nil")
	}

	err := errors.Join(
		reg.Register(ActionTypeCollect, NewCollectAction(deps.Collector)),
		reg.Register(ActionTypeEvaluate, NewEvaluateAction(deps.Evaluator)),
		reg.Register(ActionTypeExport, NewExportAction()),
		reg.Register(ActionTypeSend, NewSendAction(deps.Sender)),
		reg.Register(ActionTypeQuery, NewQueryAction()),
		reg.Register(ActionTypeSave, NewSaveAction(deps.Saver)),
		reg.Register(ActionTypeStart, StartAction{}),
		reg.Register(ActionTypeEnd, EndAction{}),
	)

	return err
}

type StartAction struct{}

func (StartAction) Execute(_ context.Context, _ registry.ActionInput) (registry.ActionOutput, error) {
	return registry.ActionOutput{Data: map[string]any{"started": true}}, nil
}

type EndAction struct{}

type pseudoEndConfig struct {
	TerminalNodeIDs []string `json:"terminal_node_ids,omitempty"`
}

func (EndAction) Execute(_ context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	var cfg pseudoEndConfig
	if len(input.Config) > 0 {
		if err := json.Unmarshal(input.Config, &cfg); err != nil {
			return registry.ActionOutput{}, fmt.Errorf("pseudo end action: decode config: %w", err)
		}
	}

	if len(cfg.TerminalNodeIDs) == 0 {
		return registry.ActionOutput{}, nil
	}

	if len(cfg.TerminalNodeIDs) == 1 {
		v, ok := input.Context.Get(cfg.TerminalNodeIDs[0])
		if !ok {
			return registry.ActionOutput{}, fmt.Errorf("pseudo end action: missing output for terminal node %q", cfg.TerminalNodeIDs[0])
		}

		return registry.ActionOutput{Data: v}, nil
	}

	out := make(map[string]any, len(cfg.TerminalNodeIDs))
	for _, id := range cfg.TerminalNodeIDs {
		v, ok := input.Context.Get(id)
		if !ok {
			return registry.ActionOutput{}, fmt.Errorf("pseudo end action: missing output for terminal node %q", id)
		}

		out[id] = v
	}

	return registry.ActionOutput{Data: out}, nil
}

type CollectAction struct {
	collector Collector
}

func NewCollectAction(collector Collector) *CollectAction {
	return &CollectAction{collector: collector}
}

type collectConfig struct {
	Cards []models.Card `json:"cards"`
}

func (a *CollectAction) Execute(ctx context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	var cfg collectConfig
	if err := json.Unmarshal(input.Config, &cfg); err != nil {
		return registry.ActionOutput{}, fmt.Errorf("collect action: decode config: %w", err)
	}

	data, err := a.collector.Collect(ctx, cfg.Cards...)
	if err != nil {
		return registry.ActionOutput{}, fmt.Errorf("collect action: collect: %w", err)
	}

	return registry.ActionOutput{Data: data}, nil
}

type EvaluateAction struct {
	evaluator Evaluator
}

func NewEvaluateAction(evaluator Evaluator) *EvaluateAction {
	return &EvaluateAction{evaluator: evaluator}
}

type evaluateConfig struct {
	Data map[string][]map[string]any `json:"data"`
	Expr string                      `json:"expr"`
}

func (a *EvaluateAction) Execute(ctx context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	var cfg evaluateConfig
	if err := json.Unmarshal(input.Config, &cfg); err != nil {
		return registry.ActionOutput{}, fmt.Errorf("evaluate action: decode config: %w", err)
	}

	approved, err := a.evaluator.Evaluate(ctx, cfg.Data, cfg.Expr)
	if err != nil {
		return registry.ActionOutput{}, fmt.Errorf("evaluate action: evaluate: %w", err)
	}

	return registry.ActionOutput{Data: map[string]any{"approved": approved}}, nil
}

type ExportAction struct{}

func NewExportAction() *ExportAction {
	return &ExportAction{}
}

type exportConfig struct {
	Data    map[string][]map[string]any `json:"data"`
	Exports []models.Export             `json:"exports"`
}

type exportOutput = []models.ExportedReport

func (a *ExportAction) Execute(_ context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	var cfg exportConfig
	if err := json.Unmarshal(input.Config, &cfg); err != nil {
		return registry.ActionOutput{}, fmt.Errorf("export action: decode config: %w", err)
	}

	var reports []*models.ExportedReport
	for _, exp := range cfg.Exports {
		r, err := exporter.Export(cfg.Data, exp)
		if err != nil {
			return registry.ActionOutput{}, fmt.Errorf("export action: export: %w", err)
		}

		reports = append(reports, r...)
	}

	return registry.ActionOutput{Data: map[string]any{"reports": reports}}, nil
}

type SendAction struct {
	sender Sender
}

func NewSendAction(sender Sender) *SendAction {
	return &SendAction{sender: sender}
}

type sendConfig struct {
	Reports    []models.ExportedReport     `json:"reports,omitempty"`
	Data       map[string][]map[string]any `json:"data"`
	Recipients []models.Recipient          `json:"recipients"`
	Approved   *bool                       `json:"approved,omitempty"`
}

func (a *SendAction) Execute(ctx context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	var cfg sendConfig
	if err := json.Unmarshal(input.Config, &cfg); err != nil {
		return registry.ActionOutput{}, fmt.Errorf("send action: decode config: %w", err)
	}

	if cfg.Approved != nil && !*cfg.Approved {
		return registry.ActionOutput{Data: map[string]any{"sent": false, "skipped": true}}, nil
	}

	if len(cfg.Reports) == 0 {
		return registry.ActionOutput{}, fmt.Errorf("send action: no reports found")
	}

	targets, err := delivery.GetTarget(cfg.Recipients...)
	if err != nil && len(targets) == 0 {
		return registry.ActionOutput{Data: map[string]any{"sent": false, "error": err.Error()}}, fmt.Errorf("send action: resolve targets: %w", err)
	}

	if len(targets) == 0 {
		return registry.ActionOutput{Data: map[string]any{"sent": false, "error": fmt.Errorf("no targets")}}, errors.New("send action: no targets")
	}

	if err := a.sender.Send(ctx, targets, cfg.Reports); err != nil {
		return registry.ActionOutput{Data: map[string]any{"sent": false, "error": err.Error()}}, fmt.Errorf("send action: send: %w", err)
	}

	return registry.ActionOutput{Data: map[string]any{"sent": true, "targets": len(targets), "reports": len(cfg.Reports)}}, nil
}

// QueryAction (std@query) loads data into an in-memory DuckDB instance and
// runs one or more named SQL queries. The results are exposed as a
// map[string][]map[string]any keyed by query name so downstream nodes can
// reference individual result-sets via "$.node_id.query_name".
//
// Config example (references resolved by engine before execution):
//
//	{
//	  "data":    "$.collect.result",        // output of std@collect node
//	  "queries": [
//	    {"name": "totals",  "sql": "SELECT category, SUM(amount) AS total FROM sales GROUP BY 1"},
//	    {"name": "summary", "sql": "SELECT COUNT(*) AS cnt FROM sales"}
//	  ]
//	}
type QueryAction struct{}

// NewQueryAction returns a QueryAction. No external dependencies are required;
// each execution creates its own in-memory DuckDB instance.
func NewQueryAction() *QueryAction {
	return &QueryAction{}
}

type queryDef struct {
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

type queryConfig struct {
	Data    map[string][]map[string]any `json:"data"`
	Queries []queryDef                  `json:"queries"`
}

func (a *QueryAction) Execute(ctx context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	var cfg queryConfig
	if err := json.Unmarshal(input.Config, &cfg); err != nil {
		return registry.ActionOutput{}, fmt.Errorf("query action: decode config: %w", err)
	}

	if len(cfg.Queries) == 0 {
		return registry.ActionOutput{}, errors.New("query action: no queries specified")
	}

	db, err := builtin.NewDB()
	if err != nil {
		return registry.ActionOutput{}, fmt.Errorf("query action: init db: %w", err)
	}

	defer db.Close() //nolint:errcheck

	if len(cfg.Data) > 0 {
		if err := db.LoadDataFromMapSlice(ctx, cfg.Data); err != nil {
			return registry.ActionOutput{}, fmt.Errorf("query action: load data: %w", err)
		}
	}

	results := make(map[string][]map[string]any, len(cfg.Queries))

	for _, q := range cfg.Queries {
		if q.Name == "" {
			return registry.ActionOutput{}, errors.New("query action: query name must not be empty")
		}

		rows, err := db.ExecuteQuery(ctx, q.SQL)
		if err != nil {
			return registry.ActionOutput{}, fmt.Errorf("query action: execute query %q: %w", q.Name, err)
		}

		results[q.Name] = rows
	}

	return registry.ActionOutput{Data: results}, nil
}

type SaveAction struct {
	db Saver
}

func NewSaveAction(s Saver) *SaveAction {
	return &SaveAction{
		db: s,
	}
}

func (a *SaveAction) Execute(ctx context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	return registry.ActionOutput{}, nil
}

package workflow_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"support_bot/internal/core/workflow/execution"
	"support_bot/internal/core/workflow/registry"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	workflow "support_bot/internal/core/workflow"
)

// --- mock action ---

type mockAction struct {
	calls atomic.Int32
	err   error
	sleep time.Duration
}

func (m *mockAction) Execute(ctx context.Context, _ registry.ActionInput) (registry.ActionOutput, error) {
	m.calls.Add(1)

	if m.sleep > 0 {
		select {
		case <-time.After(m.sleep):
		case <-ctx.Done():
			return registry.ActionOutput{}, ctx.Err()
		}
	}

	if m.err != nil {
		return registry.ActionOutput{}, m.err
	}

	return registry.ActionOutput{Data: map[string]any{"ok": true}}, nil
}

// --- helper ---

type wfBuilder struct {
	nodes []map[string]any
	edges []map[string]any
}

func (b *wfBuilder) addNode(id, typ string) *wfBuilder {
	b.nodes = append(b.nodes, map[string]any{"id": id, "type": typ})
	return b
}

func (b *wfBuilder) addEdge(from, to string) *wfBuilder {
	b.edges = append(b.edges, map[string]any{"from": from, "to": to})
	return b
}

func (b *wfBuilder) json(t *testing.T) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(map[string]any{
		"id":    "test-wf",
		"nodes": b.nodes,
		"edges": b.edges,
	})
	if err != nil {
		t.Fatal(err)
	}

	return raw
}

func newEngine(t *testing.T, actions map[string]registry.Action) *workflow.Engine {
	t.Helper()

	reg := registry.New()

	for k, v := range actions {
		err := reg.Register(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}

	return workflow.NewEngine(reg, 4, noopLogger(t), nil)
}

// --- tests ---

func TestEngine_SingleNode_Success(t *testing.T) {
	act := &mockAction{}
	engine := newEngine(t, map[string]registry.Action{"std@noop": act})

	raw := (&wfBuilder{}).addNode("n1", "std@noop").json(t)

	if err := engine.Run(context.Background(), raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if act.calls.Load() != 1 {
		t.Fatalf("want 1 call, got %d", act.calls.Load())
	}
}

func TestEngine_RunWithResult_SingleTerminal_ReturnsTerminalOutput(t *testing.T) {
	engine := newEngine(t, map[string]registry.Action{"std@noop": &mockAction{}})
	raw := (&wfBuilder{}).addNode("n1", "std@noop").json(t)

	res, err := engine.RunWithResult(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", res)
	}

	if m["ok"] != true {
		t.Fatalf("want result {ok:true}, got %#v", m)
	}
}

func TestEngine_RunWithResult_MultipleTerminals_ReturnsMapByTerminalNode(t *testing.T) {
	engine := newEngine(t, map[string]registry.Action{"std@noop": &mockAction{}})
	raw := (&wfBuilder{}).
		addNode("a", "std@noop").
		addNode("b", "std@noop").
		json(t)

	res, err := engine.RunWithResult(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", res)
	}

	if len(m) != 2 {
		t.Fatalf("want 2 terminal results, got %d", len(m))
	}

	if _, ok := m["a"]; !ok {
		t.Fatalf("terminal node a is missing in result: %#v", m)
	}

	if _, ok := m["b"]; !ok {
		t.Fatalf("terminal node b is missing in result: %#v", m)
	}
}

func TestEngine_RunWithHistory_ReturnsResultAndNodeHistory(t *testing.T) {
	engine := newEngine(t, map[string]registry.Action{"std@noop": &mockAction{}})
	raw := (&wfBuilder{}).addNode("n1", "std@noop").json(t)

	h, err := engine.RunWithHistory(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h == nil {
		t.Fatal("want non-nil history payload")
	}

	res, ok := h.Result.(map[string]any)
	if !ok || res["ok"] != true {
		t.Fatalf("unexpected result payload: %#v", h.Result)
	}

	if h.ExecutionID == "" {
		t.Fatal("execution id must be set")
	}

	if len(h.History) == 0 {
		t.Fatal("history must not be empty")
	}

	if len(h.HistoryGraph.Events) != len(h.History) {
		t.Fatalf("graph events must match flat history length: %d != %d", len(h.HistoryGraph.Events), len(h.History))
	}

	if len(h.HistoryGraph.Edges) == 0 {
		t.Fatal("history graph edges must not be empty")
	}

	seenNodeN1 := false
	for _, rec := range h.History {
		if rec.NodeID == "n1" && rec.Status == execution.StatusCompleted {
			seenNodeN1 = true
			break
		}
	}

	if !seenNodeN1 {
		t.Fatalf("history must contain completed record for n1, got %#v", h.History)
	}
}

func TestEngine_LinearChain_ExecutedInOrder(t *testing.T) {
	order := make([]string, 0, 3)
	mu := make(chan struct{}, 1)
	mu <- struct{}{}

	makeRecorder := func(id string) registry.Action {
		return &recordAction{id: id, order: &order, mu: mu}
	}

	actions := map[string]registry.Action{
		"act@a": makeRecorder("a"),
		"act@b": makeRecorder("b"),
		"act@c": makeRecorder("c"),
	}

	engine := newEngine(t, actions)
	raw := (&wfBuilder{}).
		addNode("a", "act@a").
		addNode("b", "act@b").
		addNode("c", "act@c").
		addEdge("a", "b").
		addEdge("b", "c").
		json(t)

	if err := engine.Run(context.Background(), raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("want 3 calls, got %d: %v", len(order), order)
	}

	// verify topological order: a before b, b before c
	idx := func(s string) int {
		for i, v := range order {
			if v == s {
				return i
			}
		}

		return -1
	}

	if idx("a") > idx("b") || idx("b") > idx("c") {
		t.Errorf("wrong execution order: %v", order)
	}
}

func TestEngine_ParallelBranches(t *testing.T) {
	// a → b, a → c (b and c must run in parallel)
	// If parallel: total ≈ sleep; if sequential: total ≈ 2*sleep.
	const sleep = 60 * time.Millisecond

	actions := map[string]registry.Action{
		"act@a": &mockAction{},
		"act@b": &mockAction{sleep: sleep},
		"act@c": &mockAction{sleep: sleep},
	}

	engine := newEngine(t, actions)
	raw := (&wfBuilder{}).
		addNode("a", "act@a").
		addNode("b", "act@b").
		addNode("c", "act@c").
		addEdge("a", "b").
		addEdge("a", "c").
		json(t)

	start := time.Now()

	if err := engine.Run(context.Background(), raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elapsed := time.Since(start)

	// sequential execution would take ≥ 1.8 * sleep; parallel ≈ sleep + overhead
	maxAllowed := time.Duration(float64(sleep) * 1.8)
	if elapsed > maxAllowed {
		t.Errorf("b and c should run in parallel: elapsed %v > max allowed %v (1.8× sleep)", elapsed, maxAllowed)
	}
}

func TestEngine_NodeError_WorkflowFails(t *testing.T) {
	wantErr := errors.New("boom")
	actions := map[string]registry.Action{
		"act@ok":  &mockAction{},
		"act@bad": &mockAction{err: wantErr},
	}

	engine := newEngine(t, actions)
	raw := (&wfBuilder{}).
		addNode("a", "act@ok").
		addNode("b", "act@bad").
		addEdge("a", "b").
		json(t)

	err := engine.Run(context.Background(), raw)
	if err == nil {
		t.Fatal("want error, got nil")
	}

	if !errors.Is(err, wantErr) {
		t.Fatalf("want wrapped %v, got %v", wantErr, err)
	}
}

func TestEngine_CancelContext_DownstreamDoesNotStart(t *testing.T) {
	block := make(chan struct{})
	a := newBlockingAction(block)
	b := &mockAction{}

	engine := newEngine(t, map[string]registry.Action{
		"act@a": a,
		"act@b": b,
	})

	raw := (&wfBuilder{}).
		addNode("a", "act@a").
		addNode("b", "act@b").
		addEdge("a", "b").
		json(t)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.Run(ctx, raw)
	}()

	// Wait until the first node actually starts before cancelling.
	select {
	case <-a.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for node a to start")
	}

	cancel()
	close(block)

	err := <-errCh
	if err == nil {
		t.Fatal("want cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want wrapped context canceled error, got %v", err)
	}

	if b.calls.Load() != 0 {
		t.Fatalf("downstream node should not start after cancellation, got %d calls", b.calls.Load())
	}
}

func TestEngine_FanIn_OneParentFailed_ChildSkipped(t *testing.T) {
	child := &mockAction{}
	wantErr := errors.New("boom")

	engine := newEngine(t, map[string]registry.Action{
		"act@fail": &mockAction{err: wantErr},
		// This action ignores ctx cancellation to simulate side-effecting work.
		"act@slow":  &stubbornAction{sleep: 70 * time.Millisecond},
		"act@child": child,
	})

	raw := (&wfBuilder{}).
		addNode("p1", "act@fail").
		addNode("p2", "act@slow").
		addNode("c", "act@child").
		addEdge("p1", "c").
		addEdge("p2", "c").
		json(t)

	err := engine.Run(context.Background(), raw)
	if err == nil {
		t.Fatal("want workflow error, got nil")
	}

	if !errors.Is(err, wantErr) {
		t.Fatalf("want wrapped %v, got %v", wantErr, err)
	}

	if child.calls.Load() != 0 {
		t.Fatalf("fan-in child must be skipped when any parent fails, got %d calls", child.calls.Load())
	}
}

func TestEngine_UnknownActionType_Fails(t *testing.T) {
	engine := newEngine(t, nil)

	raw := (&wfBuilder{}).addNode("n1", "std@unknown").json(t)

	err := engine.Run(context.Background(), raw)
	if err == nil {
		t.Fatal("want error for unknown action type, got nil")
	}

	if !errors.Is(err, workflow.ErrUnknownActionType) {
		t.Fatalf("want wrapped ErrUnknownActionType, got %v", err)
	}
}

func TestEngine_UnknownActionType_DoesNotRunAnyNodes(t *testing.T) {
	known := &mockAction{}
	engine := newEngine(t, map[string]registry.Action{"act@known": known})

	raw := (&wfBuilder{}).
		addNode("a", "act@known").
		addNode("b", "act@missing").
		addEdge("a", "b").
		json(t)

	err := engine.Run(context.Background(), raw)
	if err == nil {
		t.Fatal("want validation error for unknown action type, got nil")
	}

	if !errors.Is(err, workflow.ErrUnknownActionType) {
		t.Fatalf("want wrapped ErrUnknownActionType, got %v", err)
	}

	if known.calls.Load() != 0 {
		t.Fatalf("expected no side-effects before validation failure, got %d calls", known.calls.Load())
	}
}

func TestEngine_NewEngine_NilLogger(t *testing.T) {
	reg := registry.New()
	act := &mockAction{}
	reg.Register("std@noop", act)

	engine := workflow.NewEngine(reg, 1, nil, nil)
	raw := (&wfBuilder{}).addNode("n1", "std@noop").json(t)

	if err := engine.Run(context.Background(), raw); err != nil {
		t.Fatalf("unexpected error with nil logger: %v", err)
	}

	if act.calls.Load() != 1 {
		t.Fatalf("want 1 call, got %d", act.calls.Load())
	}
}

func TestEngine_NewEngine_NilRegistry_DefaultsAndRegistryAccessible(t *testing.T) {
	engine := workflow.NewEngine(nil, 0, nil, nil)

	act := &mockAction{}
	engine.Registry().Register("std@noop", act)

	raw := (&wfBuilder{}).addNode("n1", "std@noop").json(t)

	if err := engine.Run(context.Background(), raw); err != nil {
		t.Fatalf("unexpected error with nil registry: %v", err)
	}

	if act.calls.Load() != 1 {
		t.Fatalf("want 1 call, got %d", act.calls.Load())
	}
}

func TestEngine_Registry_ReturnsSamePointer(t *testing.T) {
	reg := registry.New()
	engine := workflow.NewEngine(reg, 2, noopLogger(t), nil)

	if engine.Registry() != reg {
		t.Fatal("expected Registry() to return the original registry pointer")
	}
}

func TestEngine_DuplicateEdge_FailsValidation(t *testing.T) {
	engine := newEngine(t, map[string]registry.Action{"std@noop": &mockAction{}})

	raw := (&wfBuilder{}).
		addNode("a", "std@noop").
		addNode("b", "std@noop").
		addEdge("a", "b").
		addEdge("a", "b").
		json(t)

	err := engine.Run(context.Background(), raw)
	if err == nil {
		t.Fatal("want validation error for duplicate edge, got nil")
	}
}

func TestEngine_CancelContext_NodeWaitingForWorkerDoesNotStart(t *testing.T) {
	block := make(chan struct{})
	a := newBlockingAction(block)
	b := &mockAction{}

	reg := registry.New()
	reg.Register("act@a", a)
	reg.Register("act@b", b)

	engine := workflow.NewEngine(reg, 1, noopLogger(t), nil)
	raw := (&wfBuilder{}).
		addNode("a", "act@a").
		addNode("b", "act@b").
		json(t)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.Run(ctx, raw)
	}()

	select {
	case <-a.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for node a to start")
	}

	cancel()
	close(block)

	err := <-errCh
	if err == nil {
		t.Fatal("want cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want wrapped context canceled error, got %v", err)
	}

	if b.calls.Load() != 0 {
		t.Fatalf("node waiting for worker slot should not start after cancellation, got %d calls", b.calls.Load())
	}
}

func TestEngine_InvalidJSON_Fails(t *testing.T) {
	engine := newEngine(t, nil)

	if err := engine.Run(context.Background(), json.RawMessage(`{invalid`)); err == nil {
		t.Fatal("want parse error, got nil")
	}
}

func TestEngine_CyclicWorkflow_Fails(t *testing.T) {
	engine := newEngine(t, nil)

	raw, _ := json.Marshal(map[string]any{
		"id": "cycle",
		"nodes": []map[string]any{
			{"id": "a", "type": "std@noop"},
			{"id": "b", "type": "std@noop"},
		},
		"edges": []map[string]any{
			{"from": "a", "to": "b"},
			{"from": "b", "to": "a"},
		},
	})

	if err := engine.Run(context.Background(), raw); err == nil {
		t.Fatal("want validation error for cycle, got nil")
	}
}

// --- helper action types ---

type recordAction struct {
	id    string
	order *[]string
	mu    chan struct{}
}

type blockingAction struct {
	started chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func newBlockingAction(release <-chan struct{}) *blockingAction {
	return &blockingAction{
		started: make(chan struct{}),
		release: release,
	}
}

func (b *blockingAction) Execute(ctx context.Context, _ registry.ActionInput) (registry.ActionOutput, error) {
	b.once.Do(func() { close(b.started) })

	select {
	case <-b.release:
	case <-ctx.Done():
		return registry.ActionOutput{}, ctx.Err()
	}

	return registry.ActionOutput{Data: map[string]any{"ok": true}}, nil
}

type stubbornAction struct {
	sleep time.Duration
	err   error
}

func (s *stubbornAction) Execute(_ context.Context, _ registry.ActionInput) (registry.ActionOutput, error) {
	if s.sleep > 0 {
		time.Sleep(s.sleep)
	}

	if s.err != nil {
		return registry.ActionOutput{}, s.err
	}

	return registry.ActionOutput{Data: map[string]any{"ok": true}}, nil
}

func (r *recordAction) Execute(_ context.Context, _ registry.ActionInput) (registry.ActionOutput, error) {
	<-r.mu
	*r.order = append(*r.order, r.id)
	r.mu <- struct{}{}

	return registry.ActionOutput{Data: r.id}, nil
}

// noopLogger returns a *slog.Logger that discards all output.
func noopLogger(t *testing.T) *slog.Logger {
	t.Helper()

	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

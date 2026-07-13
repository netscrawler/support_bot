package tests

import (
	"encoding/json"
	"log/slog"
	"testing"

	"support_bot/internal/core/workflow"
	"support_bot/internal/core/workflow/registry"
)

// --- helper ---

type wfBuilder struct {
	nodes []map[string]any
	edges []map[string]any
}

func newWfBuilder() *wfBuilder {
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

// noopLogger returns a *slog.Logger that discards all output.
func noopLogger(t *testing.T) *slog.Logger {
	t.Helper()

	return slog.New(slog.NewTextHandler(t.Output(), nil))
}

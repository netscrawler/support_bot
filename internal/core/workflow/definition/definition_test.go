package definition

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParse_ValidDefinition(t *testing.T) {
	raw := []byte(`{
		"id":"wf-1",
		"version":"v1",
		"nodes":[{"id":"n1","type":"std@noop","config":{"limit":10}}],
		"edges":[],
		"metadata":{"env":"test"}
	}`)

	def, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if def.ID != "wf-1" {
		t.Fatalf("want id wf-1, got %q", def.ID)
	}

	if len(def.Nodes) != 3 {
		t.Fatalf("want 3 nodes (with pseudo start/end), got %d", len(def.Nodes))
	}

	var mainNode *NodeDef
	for i := range def.Nodes {
		if def.Nodes[i].ID == "n1" {
			mainNode = &def.Nodes[i]
			break
		}
	}

	if mainNode == nil {
		t.Fatal("want node n1 to exist")
	}

	if mainNode.Type != "std@noop" {
		t.Fatalf("want node type std@noop, got %q", mainNode.Type)
	}

	var cfg map[string]any
	if err := json.Unmarshal(mainNode.Config, &cfg); err != nil {
		t.Fatalf("failed to unmarshal node config: %v", err)
	}

	if cfg["limit"] != float64(10) {
		t.Fatalf("want config limit 10, got %#v", cfg["limit"])
	}

	if len(def.Edges) != 2 {
		t.Fatalf("want 2 normalized edges, got %d", len(def.Edges))
	}
}

func TestParse_InvalidJSON_ReturnsWrappedError(t *testing.T) {
	_, err := Parse([]byte(`{"id":"wf","nodes":[`))
	if err == nil {
		t.Fatal("want parse error, got nil")
	}

	if !strings.Contains(err.Error(), "workflow/definition: parse") {
		t.Fatalf("want wrapped parse prefix, got %v", err)
	}
}

func TestParse_EmptyObject_IsAccepted(t *testing.T) {
	def, err := Parse([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error for empty object: %v", err)
	}

	if def == nil {
		t.Fatal("want non-nil definition")
	}

	if def.ID != "" {
		t.Fatalf("want empty id, got %q", def.ID)
	}
}

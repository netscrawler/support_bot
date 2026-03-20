package graph_test

import (
	"reflect"
	"testing"

	"support_bot/internal/core/workflow/definition"
	"support_bot/internal/core/workflow/graph"
)

func TestBuild_SingleNode(t *testing.T) {
	def := &definition.WorkflowDef{
		ID:    "wf",
		Nodes: []definition.NodeDef{{ID: "a", Type: "std@collect"}},
	}

	g := graph.Build(def)

	if len(g.Nodes) != 1 {
		t.Fatalf("want 1 node, got %d", len(g.Nodes))
	}

	if len(g.StartIDs) != 1 || g.StartIDs[0] != "a" {
		t.Fatalf("want start node 'a', got %v", g.StartIDs)
	}
}

func TestBuild_LinearChain(t *testing.T) {
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: "a", Type: "std@collect"},
			{ID: "b", Type: "std@export"},
			{ID: "c", Type: "std@send"},
		},
		Edges: []definition.EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	g := graph.Build(def)

	// start node
	if len(g.StartIDs) != 1 || g.StartIDs[0] != "a" {
		t.Fatalf("want single start node 'a', got %v", g.StartIDs)
	}

	// in-degrees
	if g.Nodes["a"].InDegree != 0 {
		t.Errorf("node 'a': want in-degree 0, got %d", g.Nodes["a"].InDegree)
	}

	if g.Nodes["b"].InDegree != 1 {
		t.Errorf("node 'b': want in-degree 1, got %d", g.Nodes["b"].InDegree)
	}

	if g.Nodes["c"].InDegree != 1 {
		t.Errorf("node 'c': want in-degree 1, got %d", g.Nodes["c"].InDegree)
	}

	// children
	if len(g.Nodes["a"].Children) != 1 || g.Nodes["a"].Children[0] != "b" {
		t.Errorf("node 'a' children: want [b], got %v", g.Nodes["a"].Children)
	}

	if len(g.Nodes["c"].Children) != 0 {
		t.Errorf("node 'c' should have no children, got %v", g.Nodes["c"].Children)
	}
}

func TestBuild_FanOutFanIn(t *testing.T) {
	// a → b, a → c, b → d, c → d
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: "a", Type: "std@collect"},
			{ID: "b", Type: "std@export"},
			{ID: "c", Type: "std@export"},
			{ID: "d", Type: "std@send"},
		},
		Edges: []definition.EdgeDef{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}

	g := graph.Build(def)

	if len(g.StartIDs) != 1 || g.StartIDs[0] != "a" {
		t.Fatalf("want single start node 'a', got %v", g.StartIDs)
	}

	if g.Nodes["d"].InDegree != 2 {
		t.Errorf("node 'd': want in-degree 2 (fan-in), got %d", g.Nodes["d"].InDegree)
	}

	if len(g.Nodes["a"].Children) != 2 {
		t.Errorf("node 'a': want 2 children (fan-out), got %d", len(g.Nodes["a"].Children))
	}
}

func TestBuild_MultipleStartNodes(t *testing.T) {
	// a and b are independent start nodes, both connect to c
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: "a", Type: "std@collect"},
			{ID: "b", Type: "std@collect"},
			{ID: "c", Type: "std@send"},
		},
		Edges: []definition.EdgeDef{
			{From: "a", To: "c"},
			{From: "b", To: "c"},
		},
	}

	g := graph.Build(def)

	if len(g.StartIDs) != 2 {
		t.Fatalf("want 2 start nodes, got %d: %v", len(g.StartIDs), g.StartIDs)
	}

	if !reflect.DeepEqual(g.StartIDs, []string{"a", "b"}) {
		t.Fatalf("want deterministic start order [a b], got %v", g.StartIDs)
	}

	if !reflect.DeepEqual(g.Nodes["c"].Parents, []string{"a", "b"}) {
		t.Fatalf("want deterministic parent order [a b], got %v", g.Nodes["c"].Parents)
	}

	if g.Nodes["c"].InDegree != 2 {
		t.Errorf("node 'c': want in-degree 2, got %d", g.Nodes["c"].InDegree)
	}
}

func TestBuild_DuplicateEdges_AreDeduplicated(t *testing.T) {
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: "a", Type: "std@collect"},
			{ID: "b", Type: "std@send"},
		},
		Edges: []definition.EdgeDef{
			{From: "a", To: "b"},
			{From: "a", To: "b"},
		},
	}

	g := graph.Build(def)

	if got := g.Nodes["b"].InDegree; got != 1 {
		t.Fatalf("node 'b': want deduplicated in-degree 1, got %d", got)
	}

	if !reflect.DeepEqual(g.Nodes["a"].Children, []string{"b"}) {
		t.Fatalf("node 'a': want deduplicated children [b], got %v", g.Nodes["a"].Children)
	}

	if !reflect.DeepEqual(g.Nodes["b"].Parents, []string{"a"}) {
		t.Fatalf("node 'b': want deduplicated parents [a], got %v", g.Nodes["b"].Parents)
	}
}

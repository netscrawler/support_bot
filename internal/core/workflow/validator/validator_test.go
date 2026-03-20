package validator_test

import (
	"errors"
	"testing"

	"support_bot/internal/core/workflow/definition"
	"support_bot/internal/core/workflow/validator"
)

// helpers

func makeNodes(ids ...string) []definition.NodeDef {
	nodes := []definition.NodeDef{
		{ID: definition.PseudoStartNodeID, Type: definition.PseudoStartActionType},
		{ID: definition.PseudoEndNodeID, Type: definition.PseudoEndActionType},
	}

	for _, id := range ids {
		nodes = append(nodes, definition.NodeDef{ID: id, Type: "std@collect"})
	}

	return nodes
}

func edge(from, to string) definition.EdgeDef {
	return definition.EdgeDef{From: from, To: to}
}

// --- tests ---

func TestValidate_SingleNode_OK(t *testing.T) {
	def := &definition.WorkflowDef{
		ID:    "wf1",
		Nodes: makeNodes("a"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", definition.PseudoEndNodeID),
		},
	}

	if err := validator.Validate(def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_LinearChain_OK(t *testing.T) {
	def := &definition.WorkflowDef{
		ID:    "wf2",
		Nodes: makeNodes("a", "b", "c"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", "b"), edge("b", "c"),
			edge("c", definition.PseudoEndNodeID),
		},
	}

	if err := validator.Validate(def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_FanOutFanIn_OK(t *testing.T) {
	// a → b, a → c, b → d, c → d
	def := &definition.WorkflowDef{
		ID:    "wf3",
		Nodes: makeNodes("a", "b", "c", "d"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", "b"), edge("a", "c"),
			edge("b", "d"), edge("c", "d"),
			edge("d", definition.PseudoEndNodeID),
		},
	}

	if err := validator.Validate(def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyNodes(t *testing.T) {
	def := &definition.WorkflowDef{ID: "wf"}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrEmptyNodes) {
		t.Fatalf("want ErrEmptyNodes, got %v", err)
	}
}

func TestValidate_NilDefinition(t *testing.T) {
	if err := validator.Validate(nil); !errors.Is(err, validator.ErrNilWorkflowDef) {
		t.Fatalf("want ErrNilWorkflowDef, got %v", err)
	}
}

func TestValidate_DuplicateID(t *testing.T) {
	def := &definition.WorkflowDef{
		ID:    "wf",
		Nodes: makeNodes("a", "a"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", definition.PseudoEndNodeID),
		},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrDuplicateNodeID) {
		t.Fatalf("want ErrDuplicateNodeID, got %v", err)
	}
}

func TestValidate_EmptyNodeID(t *testing.T) {
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: definition.PseudoStartNodeID, Type: definition.PseudoStartActionType},
			{ID: definition.PseudoEndNodeID, Type: definition.PseudoEndActionType},
			{ID: "", Type: "std@collect"},
		},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrEmptyNodeID) {
		t.Fatalf("want ErrEmptyNodeID, got %v", err)
	}
}

func TestValidate_EmptyNodeType(t *testing.T) {
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: definition.PseudoStartNodeID, Type: definition.PseudoStartActionType},
			{ID: definition.PseudoEndNodeID, Type: definition.PseudoEndActionType},
			{ID: "a", Type: ""},
		},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrEmptyNodeType) {
		t.Fatalf("want ErrEmptyNodeType, got %v", err)
	}
}

func TestValidate_UnknownEdgeFrom(t *testing.T) {
	def := &definition.WorkflowDef{
		ID:    "wf",
		Nodes: makeNodes("a"),
		Edges: []definition.EdgeDef{
			edge("missing", "a"),
			edge("a", definition.PseudoEndNodeID),
		},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrUnknownEdgeRef) {
		t.Fatalf("want ErrUnknownEdgeRef, got %v", err)
	}
}

func TestValidate_UnknownEdgeTo(t *testing.T) {
	def := &definition.WorkflowDef{
		ID:    "wf",
		Nodes: makeNodes("a"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", "missing"),
		},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrUnknownEdgeRef) {
		t.Fatalf("want ErrUnknownEdgeRef, got %v", err)
	}
}

func TestValidate_DuplicateEdge(t *testing.T) {
	def := &definition.WorkflowDef{
		ID:    "wf",
		Nodes: makeNodes("a", "b"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", "b"),
			edge("a", "b"),
			edge("b", definition.PseudoEndNodeID),
		},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrDuplicateEdge) {
		t.Fatalf("want ErrDuplicateEdge, got %v", err)
	}
}

func TestValidate_CycleDetected(t *testing.T) {
	// a → b → c → a  (cycle)
	def := &definition.WorkflowDef{
		ID:    "wf",
		Nodes: makeNodes("a", "b", "c"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", "b"), edge("b", "c"), edge("c", "a"),
			edge("c", definition.PseudoEndNodeID),
		},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrCycleDetected) {
		t.Fatalf("want ErrCycleDetected, got %v", err)
	}
}

func TestValidate_AllNodesHaveParents_NoStart(t *testing.T) {
	// Two nodes pointing at each other — no in-degree-0 node.
	def := &definition.WorkflowDef{
		ID:    "wf",
		Nodes: makeNodes("a", "b"),
		Edges: []definition.EdgeDef{
			edge(definition.PseudoStartNodeID, "a"),
			edge("a", "b"), edge("b", "a"),
			edge("b", definition.PseudoEndNodeID),
		},
	}

	err := validator.Validate(def)
	if !errors.Is(err, validator.ErrCycleDetected) {
		t.Fatalf("want ErrCycleDetected, got %v", err)
	}
}

func TestValidate_MissingPseudoStart(t *testing.T) {
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: definition.PseudoEndNodeID, Type: definition.PseudoEndActionType},
			{ID: "a", Type: "std@collect"},
		},
		Edges: []definition.EdgeDef{edge("a", definition.PseudoEndNodeID)},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrMissingPseudoStart) {
		t.Fatalf("want ErrMissingPseudoStart, got %v", err)
	}
}

func TestValidate_MissingPseudoEnd(t *testing.T) {
	def := &definition.WorkflowDef{
		ID: "wf",
		Nodes: []definition.NodeDef{
			{ID: definition.PseudoStartNodeID, Type: definition.PseudoStartActionType},
			{ID: "a", Type: "std@collect"},
		},
		Edges: []definition.EdgeDef{edge(definition.PseudoStartNodeID, "a")},
	}

	if err := validator.Validate(def); !errors.Is(err, validator.ErrMissingPseudoEnd) {
		t.Fatalf("want ErrMissingPseudoEnd, got %v", err)
	}
}

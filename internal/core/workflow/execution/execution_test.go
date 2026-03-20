package execution_test

import (
	"errors"
	"sort"
	"testing"

	"support_bot/internal/core/workflow/execution"
	"support_bot/internal/core/workflow/graph"
)

func TestExecutionSkipAllWaiting_MakesExecutionComplete(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"a": {ID: "a", Type: "act@a"},
			"b": {ID: "b", Type: "act@b"},
		},
		StartIDs: []string{"a", "b"},
	}

	exec := execution.New("wf", g)
	if exec.IsComplete() {
		t.Fatal("new execution must not be complete")
	}

	cause := errors.New("cancelled")
	skipped := exec.SkipAllWaiting(cause)
	if skipped != 2 {
		t.Fatalf("want 2 skipped nodes, got %d", skipped)
	}

	if !exec.IsComplete() {
		t.Fatal("execution must be complete after skipping all waiting nodes")
	}

	for _, id := range []string{"a", "b"} {
		st := exec.GetState(id)
		if st == nil {
			t.Fatalf("missing state for node %s", id)
		}
		if st.Status != execution.StatusSkipped {
			t.Fatalf("node %s: want skipped, got %s", id, st.Status)
		}
		if !errors.Is(st.Err, cause) {
			t.Fatalf("node %s: want wrapped cause %v, got %v", id, cause, st.Err)
		}
	}
}

func TestExecutionSetState_CompletedWithNilOutput_StoresInContext(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"n1": {ID: "n1", Type: "act@n1"},
		},
		StartIDs: []string{"n1"},
	}

	exec := execution.New("wf", g)
	exec.SetState("n1", execution.StatusCompleted, nil, nil)

	if _, ok := exec.Context.Get("n1"); !ok {
		t.Fatal("want context key for n1 to exist even when output is nil")
	}

	resolved, err := exec.Context.Resolve("$.n1")
	if err != nil {
		t.Fatalf("want nil output to be resolvable, got error: %v", err)
	}

	if resolved != nil {
		t.Fatalf("want resolved value to be nil, got %#v", resolved)
	}
}

func TestExecutionSkipIfWaiting_TransitionsOnlyFromWaiting(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"n1": {ID: "n1", Type: "act@n1"},
			"n2": {ID: "n2", Type: "act@n2"},
		},
		StartIDs: []string{"n1", "n2"},
	}

	exec := execution.New("wf", g)
	cause := errors.New("cancelled")

	if !exec.SkipIfWaiting("n1", cause) {
		t.Fatal("want n1 to transition from waiting to skipped")
	}

	st := exec.GetState("n1")
	if st == nil || st.Status != execution.StatusSkipped {
		t.Fatalf("want n1 skipped, got %#v", st)
	}

	if !errors.Is(st.Err, cause) {
		t.Fatalf("want wrapped cause %v, got %v", cause, st.Err)
	}

	if exec.SkipIfWaiting("n1", errors.New("another cause")) {
		t.Fatal("want false when skipping already terminal node")
	}

	exec.SetState("n2", execution.StatusRunning, nil, nil)
	if exec.SkipIfWaiting("n2", cause) {
		t.Fatal("want false when node is running")
	}

	if exec.SkipIfWaiting("unknown", cause) {
		t.Fatal("want false for unknown node")
	}
}

func TestExecutionHasFailedAndFailedNodes(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"a": {ID: "a", Type: "act@a"},
			"b": {ID: "b", Type: "act@b"},
			"c": {ID: "c", Type: "act@c"},
		},
		StartIDs: []string{"a", "b", "c"},
	}

	exec := execution.New("wf", g)
	if exec.HasFailed() {
		t.Fatal("new execution must not be failed")
	}

	exec.SetState("a", execution.StatusCompleted, nil, nil)
	exec.SetState("b", execution.StatusFailed, nil, errors.New("boom b"))
	exec.SetState("c", execution.StatusFailed, nil, errors.New("boom c"))

	if !exec.HasFailed() {
		t.Fatal("want HasFailed=true when at least one node failed")
	}

	failed := exec.FailedNodes()
	sort.Strings(failed)

	if len(failed) != 2 || failed[0] != "b" || failed[1] != "c" {
		t.Fatalf("want failed nodes [b c], got %v", failed)
	}

	if !exec.IsComplete() {
		t.Fatal("execution with completed/failed nodes must be complete")
	}
}

func TestExecutionHistory_RecordsTransitionsAndErrors(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"a": {ID: "a", Type: "act@a"},
		},
		StartIDs: []string{"a"},
	}

	exec := execution.New("wf", g)
	errBoom := errors.New("boom")

	exec.SetState("a", execution.StatusRunning, nil, nil)
	exec.SetState("a", execution.StatusFailed, nil, errBoom)

	h := exec.History()
	if len(h) != 2 {
		t.Fatalf("want 2 history records, got %d", len(h))
	}

	if h[0].NodeID != "a" || h[0].Status != execution.StatusRunning {
		t.Fatalf("unexpected first record: %#v", h[0])
	}

	if h[1].NodeID != "a" || h[1].Status != execution.StatusFailed {
		t.Fatalf("unexpected second record: %#v", h[1])
	}

	if h[1].Error != "boom" {
		t.Fatalf("want history error 'boom', got %q", h[1].Error)
	}

	if h[0].UpdatedAt.IsZero() || h[1].UpdatedAt.IsZero() {
		t.Fatalf("history timestamps must be set: %#v", h)
	}
}

func TestExecutionHistoryGraph_HasSequenceAndDependencyEdges(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"a": {ID: "a", Type: "act@a", Children: []string{"b"}},
			"b": {ID: "b", Type: "act@b", Parents: []string{"a"}},
		},
		StartIDs: []string{"a"},
	}

	exec := execution.New("wf", g)
	exec.SetState("a", execution.StatusRunning, nil, nil)
	exec.SetState("a", execution.StatusCompleted, map[string]any{"ok": true}, nil)
	exec.SetState("b", execution.StatusRunning, nil, nil)
	exec.SetState("b", execution.StatusCompleted, map[string]any{"ok": true}, nil)

	hg := exec.HistoryGraph()
	if len(hg.Events) != 4 {
		t.Fatalf("want 4 history events, got %d", len(hg.Events))
	}

	var hasSeqA, hasSeqB, hasDep bool
	for _, e := range hg.Edges {
		switch e.Type {
		case execution.HistoryEdgeNodeSequence:
			if e.FromEventID == 1 && e.ToEventID == 2 {
				hasSeqA = true
			}
			if e.FromEventID == 3 && e.ToEventID == 4 {
				hasSeqB = true
			}
		case execution.HistoryEdgeDependency:
			if e.FromEventID == 2 && e.ToEventID == 3 {
				hasDep = true
			}
		}
	}

	if !hasSeqA || !hasSeqB || !hasDep {
		t.Fatalf("unexpected history graph edges: %#v", hg.Edges)
	}
}

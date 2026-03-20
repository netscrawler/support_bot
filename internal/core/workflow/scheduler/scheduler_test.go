package scheduler

import (
	"errors"
	"testing"

	"support_bot/internal/core/workflow/execution"
	"support_bot/internal/core/workflow/graph"
)

func TestStart_EnqueuesStartNodesInSortedOrder(t *testing.T) {
	exec := execution.New("wf", &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"a": {ID: "a", Type: "act@a"},
			"b": {ID: "b", Type: "act@b"},
		},
		StartIDs: []string{"b", "a"},
	})

	s := New(exec, 1)
	s.Start()

	first := <-s.ReadyCh()
	second := <-s.ReadyCh()

	if first != "a" || second != "b" {
		t.Fatalf("want [a b], got [%s %s]", first, second)
	}
}

func TestComplete_EnqueuesChildAfterAllParentsCompleted(t *testing.T) {
	exec := execution.New("wf", &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"p1": {ID: "p1", Type: "act@p1", Children: []string{"c"}},
			"p2": {ID: "p2", Type: "act@p2", Children: []string{"c"}},
			"c":  {ID: "c", Type: "act@c", InDegree: 2, Parents: []string{"p1", "p2"}},
		},
		StartIDs: []string{"p1", "p2"},
	})

	s := New(exec, 1)

	exec.SetState("p1", execution.StatusCompleted, nil, nil)
	s.Complete("p1")

	select {
	case id := <-s.ReadyCh():
		t.Fatalf("child must not be ready after one parent only, got %s", id)
	default:
	}

	exec.SetState("p2", execution.StatusCompleted, nil, nil)
	s.Complete("p2")

	select {
	case id := <-s.ReadyCh():
		if id != "c" {
			t.Fatalf("want child c, got %s", id)
		}
	default:
		t.Fatal("want child c to be enqueued after all parents complete")
	}
}

func TestComplete_SkipsChildWhenAnyParentNotCompleted(t *testing.T) {
	exec := execution.New("wf", &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"p1": {ID: "p1", Type: "act@p1", Children: []string{"c"}},
			"p2": {ID: "p2", Type: "act@p2", Children: []string{"c"}},
			"c":  {ID: "c", Type: "act@c", InDegree: 2, Parents: []string{"p1", "p2"}},
		},
		StartIDs: []string{"p1", "p2"},
	})

	s := New(exec, 1)

	upstreamErr := errors.New("upstream failed")
	exec.SetState("p1", execution.StatusFailed, nil, upstreamErr)
	s.Complete("p1")

	exec.SetState("p2", execution.StatusCompleted, nil, nil)
	s.Complete("p2")

	st := exec.GetState("c")
	if st == nil {
		t.Fatal("missing state for child node c")
	}

	if st.Status != execution.StatusSkipped {
		t.Fatalf("want skipped status, got %s", st.Status)
	}

	if !errors.Is(st.Err, errUpstreamNotCompleted) {
		t.Fatalf("want upstream-not-completed error, got %v", st.Err)
	}
}

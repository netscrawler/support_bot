package executor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"support_bot/internal/core/workflow/execution"
	"support_bot/internal/core/workflow/graph"
	"support_bot/internal/core/workflow/registry"
)

type panicAction struct{}

type captureConfigAction struct {
	called int
	last   map[string]any
}

func (a *captureConfigAction) Execute(_ context.Context, in registry.ActionInput) (registry.ActionOutput, error) {
	a.called++

	if err := json.Unmarshal(in.Config, &a.last); err != nil {
		return registry.ActionOutput{}, err
	}

	return registry.ActionOutput{Data: map[string]any{"ok": true}}, nil
}

func (panicAction) Execute(context.Context, registry.ActionInput) (registry.ActionOutput, error) {
	panic("boom")
}

func TestRunNode_MissingRuntimeNode_FailsGracefully(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"dangling": nil,
		},
		StartIDs: []string{"dangling"},
	}

	exec := execution.New("wf", g)
	ex := New(exec, nil, nil, 1, nil)

	err := ex.runNode(context.Background(), "dangling")
	if err == nil {
		t.Fatal("want error for missing runtime node, got nil")
	}

	st := exec.GetState("dangling")
	if st == nil {
		t.Fatal("expected state to exist for dangling node")
	}

	if st.Status != execution.StatusFailed {
		t.Fatalf("want failed status, got %s", st.Status)
	}

	if !errors.Is(st.Err, err) && st.Err == nil {
		t.Fatalf("want node state error to be recorded, got %v", st.Err)
	}
}

func TestRunNode_ActionPanic_MarksNodeFailed(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"n1": {ID: "n1", Type: "panic@action"},
		},
		StartIDs: []string{"n1"},
	}

	exec := execution.New("wf", g)
	reg := registry.New()
	reg.Register("panic@action", panicAction{})

	ex := New(exec, reg, nil, 1, nil)
	err := ex.runNode(context.Background(), "n1")
	if err == nil {
		t.Fatal("want error when action panics, got nil")
	}

	st := exec.GetState("n1")
	if st == nil {
		t.Fatal("expected state for node n1")
	}

	if st.Status != execution.StatusFailed {
		t.Fatalf("want failed status, got %s", st.Status)
	}

	if st.Err == nil {
		t.Fatal("want node error to be captured")
	}
}

func TestRunNode_ConfigReferences_AreResolvedBeforeAction(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"child": {
				ID:     "child",
				Type:   "act@capture",
				Config: json.RawMessage(`{"token":"$.parent.token"}`),
			},
		},
		StartIDs: []string{"child"},
	}

	exec := execution.New("wf", g)
	exec.Context.Set("parent", map[string]any{"token": "abc"})

	reg := registry.New()
	act := &captureConfigAction{}
	reg.Register("act@capture", act)

	ex := New(exec, reg, nil, 1, nil)
	if err := ex.runNode(context.Background(), "child"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if act.called != 1 {
		t.Fatalf("want action called once, got %d", act.called)
	}

	if act.last["token"] != "abc" {
		t.Fatalf("want resolved token=abc, got %#v", act.last["token"])
	}
}

func TestRunNode_ConfigReferences_ErrorMarksNodeFailed(t *testing.T) {
	g := &graph.RuntimeGraph{
		Nodes: map[string]*graph.RuntimeNode{
			"child": {
				ID:     "child",
				Type:   "act@capture",
				Config: json.RawMessage(`{"token":"$.missing.token"}`),
			},
		},
		StartIDs: []string{"child"},
	}

	exec := execution.New("wf", g)
	reg := registry.New()
	act := &captureConfigAction{}
	reg.Register("act@capture", act)

	ex := New(exec, reg, nil, 1, nil)
	err := ex.runNode(context.Background(), "child")
	if err == nil {
		t.Fatal("want error for unresolved config reference")
	}

	if act.called != 0 {
		t.Fatalf("action must not run when config resolution fails, got %d calls", act.called)
	}

	st := exec.GetState("child")
	if st == nil || st.Status != execution.StatusFailed {
		t.Fatalf("want failed node state, got %#v", st)
	}
}

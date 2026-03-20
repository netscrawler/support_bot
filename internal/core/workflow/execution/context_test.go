package execution_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"support_bot/internal/core/workflow/execution"
)

type sampleOutput struct {
	Name string
	Age  int
}

type nestedSampleOutput struct {
	Profile sampleOutput
}

func TestExecutionContextResolve_MapStringInterface(t *testing.T) {
	ctx := execution.NewExecutionContext()
	ctx.Set("n1", map[string]interface{}{"value": 42})

	got, err := ctx.Resolve("$.n1.value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 42 {
		t.Fatalf("want 42, got %#v", got)
	}
}

func TestExecutionContextResolve_StructField(t *testing.T) {
	ctx := execution.NewExecutionContext()
	ctx.Set("n1", sampleOutput{Name: "alice", Age: 30})

	got, err := ctx.Resolve("$.n1.Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "alice" {
		t.Fatalf("want alice, got %#v", got)
	}
}

func TestExecutionContextResolve_NestedMapPath(t *testing.T) {
	ctx := execution.NewExecutionContext()
	ctx.Set("n1", map[string]any{
		"profile": map[string]any{
			"name": "alice",
		},
	})

	got, err := ctx.Resolve("$.n1.profile.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "alice" {
		t.Fatalf("want alice, got %#v", got)
	}
}

func TestExecutionContextResolve_NestedStructPath(t *testing.T) {
	ctx := execution.NewExecutionContext()
	ctx.Set("n1", nestedSampleOutput{Profile: sampleOutput{Name: "alice", Age: 30}})

	got, err := ctx.Resolve("$.n1.Profile.Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "alice" {
		t.Fatalf("want alice, got %#v", got)
	}
}

func TestExecutionContextResolve_PointerPath(t *testing.T) {
	ctx := execution.NewExecutionContext()
	ctx.Set("n1", &nestedSampleOutput{Profile: sampleOutput{Name: "alice", Age: 30}})

	got, err := ctx.Resolve("$.n1.Profile.Age")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 30 {
		t.Fatalf("want 30, got %#v", got)
	}
}

func TestExecutionContextResolve_InvalidReference(t *testing.T) {
	ctx := execution.NewExecutionContext()

	if _, err := ctx.Resolve("$."); err == nil {
		t.Fatal("want error for empty node id")
	}

	if _, err := ctx.Resolve("$.n1."); err == nil {
		t.Fatal("want error for empty field")
	}
}

func TestExecutionContextResolveConfig_ReplacesReferencesRecursively(t *testing.T) {
	ctx := execution.NewExecutionContext()
	ctx.Set("producer", map[string]any{
		"token": "abc",
		"meta":  map[string]any{"retries": 3},
	})

	raw := json.RawMessage(`{
		"auth": "$.producer.token",
		"meta": "$.producer.meta",
		"list": ["$.producer.token", {"nested": "$.producer.meta.retries"}],
		"plain": "keep-me"
	}`)

	resolvedRaw, err := ctx.ResolveConfig(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(resolvedRaw, &got); err != nil {
		t.Fatalf("unmarshal resolved config: %v", err)
	}

	if got["auth"] != "abc" {
		t.Fatalf("want auth=abc, got %#v", got["auth"])
	}

	wantMeta := map[string]any{"retries": float64(3)}
	if !reflect.DeepEqual(got["meta"], wantMeta) {
		t.Fatalf("want meta=%#v, got %#v", wantMeta, got["meta"])
	}

	list, ok := got["list"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("want list with 2 items, got %#v", got["list"])
	}

	if list[0] != "abc" {
		t.Fatalf("want first list item=abc, got %#v", list[0])
	}

	nested, ok := list[1].(map[string]any)
	if !ok || nested["nested"] != float64(3) {
		t.Fatalf("want nested value=3, got %#v", list[1])
	}

	if got["plain"] != "keep-me" {
		t.Fatalf("want plain value unchanged, got %#v", got["plain"])
	}
}

func TestExecutionContextResolveConfig_MissingReferenceReturnsError(t *testing.T) {
	ctx := execution.NewExecutionContext()
	raw := json.RawMessage(`{"auth": "$.missing.token"}`)

	if _, err := ctx.ResolveConfig(raw); err == nil {
		t.Fatal("want error for missing reference")
	}
}

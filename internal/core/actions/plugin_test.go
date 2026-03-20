package actions

import (
	"context"
	"encoding/json"
	"errors"
	"support_bot/internal/core/workflow/registry"
	"testing"

	plugins "support_bot/internal/plugin"
)

type stubRunner struct {
	out       []byte
	err       error
	gotParams map[string]any
}

func (s *stubRunner) Execute(_ context.Context, params map[string]any) ([]byte, error) {
	s.gotParams = params

	return s.out, s.err
}

type stubManager struct {
	newPluginFn func(ctx context.Context, name string, version *string) (*plugins.LuaPlugin, error)
}

func (s stubManager) NewPlugin(ctx context.Context, name string, version *string) (*plugins.LuaPlugin, error) {
	if s.newPluginFn == nil {
		return &plugins.LuaPlugin{}, nil
	}

	return s.newPluginFn(ctx, name, version)
}

func TestResolvePluginNameVer(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		name, ver, err := resolvePluginNameVer("extend_phone@v1.2.3")
		if err != nil {
			t.Fatalf("resolvePluginNameVer returned error: %v", err)
		}

		if name != "extend_phone" || ver != "v1.2.3" {
			t.Fatalf("unexpected output: name=%q ver=%q", name, ver)
		}
	})

	t.Run("invalid old format", func(t *testing.T) {
		t.Parallel()

		_, _, err := resolvePluginNameVer("extend_phone@v.1.2.3")
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestPluginActionExecute_DecodesOutput(t *testing.T) {
	t.Parallel()

	runner := &stubRunner{out: []byte(`{"ok":true,"count":2}`)}
	action := &PluginAction{plugin: runner}

	in := registry.ActionInput{Config: json.RawMessage(`{"arg":"value"}`)}
	out, err := action.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if runner.gotParams["arg"] != "value" {
		t.Fatalf("plugin params were not passed, got: %#v", runner.gotParams)
	}

	decoded, ok := out.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map output, got %T", out.Data)
	}

	if decoded["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", decoded["ok"])
	}
}

func TestPluginActionExecute_InvalidOutputJSON(t *testing.T) {
	t.Parallel()

	action := &PluginAction{plugin: &stubRunner{out: []byte("not-json")}}
	_, err := action.Execute(context.Background(), registry.ActionInput{})
	if err == nil {
		t.Fatal("expected decode output error")
	}
}

func TestRegisterPluginsFromRaw(t *testing.T) {
	t.Parallel()

	reg := registry.New()
	calls := 0

	manager := stubManager{
		newPluginFn: func(_ context.Context, name string, version *string) (*plugins.LuaPlugin, error) {
			calls++
			if name != "extend_phone" {
				t.Fatalf("unexpected plugin name: %q", name)
			}
			if version == nil || *version != "v1.2.3" {
				t.Fatalf("unexpected plugin version: %v", version)
			}

			return &plugins.LuaPlugin{}, nil
		},
	}

	raw := json.RawMessage(`{
		"id": "wf-1",
		"nodes": [
			{"id":"collect","type":"std@collect"},
			{"id":"plug-1","type":"plugin@extend_phone@v1.2.3"},
			{"id":"plug-2","type":"plugin@extend_phone@v1.2.3"}
		],
		"edges": []
	}`)

	if err := RegisterPluginsFromRaw(context.Background(), reg, manager, raw); err != nil {
		t.Fatalf("RegisterPluginsFromRaw returned error: %v", err)
	}

	if !reg.Has("plugin@extend_phone@v1.2.3") {
		t.Fatal("expected plugin action to be registered")
	}

	if calls != 1 {
		t.Fatalf("expected one plugin construction for duplicated type, got %d", calls)
	}
}

func TestRegisterPluginsFromDefinition_Validation(t *testing.T) {
	t.Parallel()

	err := RegisterPluginsFromDefinition(context.Background(), nil, stubManager{}, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}

	err = RegisterPluginsFromDefinition(context.Background(), registry.New(), nil, nil)
	if err == nil {
		t.Fatal("expected manager validation error")
	}
}

func TestNewPluginAction_NilPlugin(t *testing.T) {
	t.Parallel()

	manager := stubManager{
		newPluginFn: func(_ context.Context, _ string, _ *string) (*plugins.LuaPlugin, error) {
			return nil, nil
		},
	}

	_, err := NewPluginAction(context.Background(), "plug@v1", manager)
	if err == nil {
		t.Fatal("expected error for nil plugin")
	}
}

func TestPluginActionExecute_PropagatesPluginError(t *testing.T) {
	t.Parallel()

	action := &PluginAction{plugin: &stubRunner{err: errors.New("boom")}}
	_, err := action.Execute(context.Background(), registry.ActionInput{})
	if err == nil {
		t.Fatal("expected plugin execution error")
	}
}

package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)
	require.NoError(t, err)
	require.NotNil(t, manager, "expected manager to be created")

	assert.Empty(t, manager.plugins, "expected empty plugins map")
}

func TestLoadPlugin(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)
	require.NoError(t, err)

	// create temporary test plugin
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "test.lua")
	pluginContent := `
plugin = {
    name = "test_plugin",
    version = "1.0.0",
    description = "test",
    author = "test"
}

function plugin.init(config)
    return true, nil
end

function plugin.fetch_data(params)
    local json = require("json")
    return json.encode({test = "data"}), nil
end

function plugin.validate(params)
    return true, nil
end

function plugin.cleanup()
end
`

	if err := os.WriteFile(pluginPath, []byte(pluginContent), 0o600); err != nil {
		t.Fatalf("failed to write test plugin: %v", err)
	}

	// test loading
	if err := manager.LoadPlugin(pluginPath); err != nil {
		t.Fatalf("failed to load plugin: %v", err)
	}

	// verify plugin was loaded
	plugin, err := manager.GetPlugin("test_plugin")
	if err != nil {
		t.Fatalf("failed to get loaded plugin: %v", err)
	}

	if plugin.Name() != "test_plugin" {
		t.Errorf("expected plugin name 'test_plugin', got '%s'", plugin.Name())
	}

	if plugin.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", plugin.Version())
	}
}

func TestLoadPlugin_Duplicate(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)

	require.NoError(t, err)

	// create temporary test plugin
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "test.lua")
	pluginContent := `
plugin = {
    name = "duplicate",
    version = "1.0.0"
}
function plugin.init(config) return true, nil end
function plugin.fetch_data(params) return "{}", nil end
function plugin.validate(params) return true, nil end
function plugin.cleanup() end
`

	if err := os.WriteFile(pluginPath, []byte(pluginContent), 0o600); err != nil {
		t.Fatalf("failed to write test plugin: %v", err)
	}

	// load first time
	if err := manager.LoadPlugin(pluginPath); err != nil {
		t.Fatalf("failed to load plugin first time: %v", err)
	}

	// try to load again - should fail
	err = manager.LoadPlugin(pluginPath)
	if err == nil {
		t.Error("expected error when loading duplicate plugin")
	}
}

func TestUnloadPlugin(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)
	require.NoError(t, err)

	// create and load test plugin
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "test.lua")
	pluginContent := `
plugin = {name = "unload_test", version = "1.0.0"}
function plugin.init(config) return true, nil end
function plugin.fetch_data(params) return "{}", nil end
function plugin.validate(params) return true, nil end
function plugin.cleanup() end
`

	if err := os.WriteFile(pluginPath, []byte(pluginContent), 0o600); err != nil {
		t.Fatalf("failed to write test plugin: %v", err)
	}

	if err := manager.LoadPlugin(pluginPath); err != nil {
		t.Fatalf("failed to load plugin: %v", err)
	}

	// unload plugin
	if err := manager.UnloadPlugin("unload_test"); err != nil {
		t.Fatalf("failed to unload plugin: %v", err)
	}

	// verify plugin is gone
	_, err = manager.GetPlugin("unload_test")
	if err == nil {
		t.Error("expected error when getting unloaded plugin")
	}
}

func TestGetPlugin_NotFound(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)
	require.NoError(t, err)

	_, err = manager.GetPlugin("nonexistent")
	if err == nil {
		t.Error("expected error when getting nonexistent plugin")
	}
}

func TestListPlugins(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)
	require.NoError(t, err)

	// create multiple test plugins
	tmpDir := t.TempDir()

	for i := 1; i <= 3; i++ {
		pluginPath := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".lua")

		pluginContent := `
plugin = {name = "plugin` + string(rune('0'+i)) + `", version = "1.0.0"}
function plugin.init(config) return true, nil end
function plugin.fetch_data(params) return "{}", nil end
function plugin.validate(params) return true, nil end
function plugin.cleanup() end
`
		if err := os.WriteFile(pluginPath, []byte(pluginContent), 0o600); err != nil {
			t.Fatalf("failed to write test plugin %d: %v", i, err)
		}

		if err := manager.LoadPlugin(pluginPath); err != nil {
			t.Fatalf("failed to load plugin %d: %v", i, err)
		}
	}

	// list all plugins
	infos := manager.ListPlugins()
	if len(infos) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(infos))
	}
}

func TestPluginInit(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)
	require.NoError(t, err)

	// create test plugin
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "init_test.lua")
	pluginContent := `
plugin = {name = "init_test", version = "1.0.0"}
function plugin.init(config)
    plugin.test_value = config.test_key
    return true, nil
end
function plugin.fetch_data(params) 
    local json = require("json")
    return json.encode({value = plugin.test_value}), nil
end
function plugin.validate(params) return true, nil end
function plugin.cleanup() end
`

	if err := os.WriteFile(pluginPath, []byte(pluginContent), 0o600); err != nil {
		t.Fatalf("failed to write test plugin: %v", err)
	}

	if err := manager.LoadPlugin(pluginPath); err != nil {
		t.Fatalf("failed to load plugin: %v", err)
	}

	plugin, err := manager.GetPlugin("init_test")
	if err != nil {
		t.Fatalf("failed to get plugin: %v", err)
	}

	// test init with config
	config := map[string]any{
		"test_key": "test_value",
	}

	if err := plugin.Init(config); err != nil {
		t.Fatalf("failed to init plugin: %v", err)
	}
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Enable: true,
	}

	manager, err := NewManager(cfg)
	require.NoError(t, err)

	// load some plugins
	tmpDir := t.TempDir()

	for i := 1; i <= 2; i++ {
		pluginPath := filepath.Join(tmpDir, "shutdown"+string(rune('0'+i))+".lua")

		pluginContent := `
plugin = {name = "shutdown` + string(rune('0'+i)) + `", version = "1.0.0"}
function plugin.init(config) return true, nil end
function plugin.fetch_data(params) return "{}", nil end
function plugin.validate(params) return true, nil end
function plugin.cleanup() end
`
		if err := os.WriteFile(pluginPath, []byte(pluginContent), 0o600); err != nil {
			t.Fatalf("failed to write test plugin: %v", err)
		}

		if err := manager.LoadPlugin(pluginPath); err != nil {
			t.Fatalf("failed to load plugin: %v", err)
		}
	}

	// shutdown
	if err := manager.Shutdown(); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// verify all plugins are gone
	infos := manager.ListPlugins()
	if len(infos) != 0 {
		t.Errorf("expected 0 plugins after shutdown, got %d", len(infos))
	}
}

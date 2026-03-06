package plugins

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBasicLuaPluginCreation tests that we can create and use a basic Lua plugin
func TestBasicLuaPluginCreation(t *testing.T) {
	t.Parallel()

	script := `
plugin = {
    name = "basic_test",
    version = "1.0.0",
    description = "basic test plugin",
    author = "test",

    init = function(config)
        return true, nil
    end,

    execute = function(params)
        return {result = "ok"}, nil
    end,

    validate = function(params)
        return true, nil
    end,

    cleanup = function()
    end
}
`

	plugin, err := NewLuaPluginWithConfigFromString(script, DefaultRuntimeConfig(), nil)
	require.NoError(t, err, "failed to create plugin")
	require.NotNil(t, plugin)

	// Test Init
	err = plugin.Init(map[string]any{"test": "value"})
	require.NoError(t, err, "failed to init plugin")

	// Test Execute
	data, err := plugin.Execute(context.Background(), map[string]any{"input": "test"})
	require.NoError(t, err, "failed to execute plugin")
	require.NotNil(t, data)
	t.Logf("Execute result: %s", string(data))

	// Test Validate
	err = plugin.Validate(map[string]any{"input": "test"})
	require.NoError(t, err, "failed to validate params")

	// Test Cleanup
	err = plugin.Cleanup()
	require.NoError(t, err, "failed to cleanup plugin")
}

package plugins

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPluginLoader struct {
	loadByNameFn           func(ctx context.Context, name string) (LuaPluginDTO, error)
	loadByNameAndVersionFn func(ctx context.Context, name, version string) (LuaPluginDTO, error)
}

func (f *mockPluginLoader) LoadByName(ctx context.Context, name string) (LuaPluginDTO, error) {
	if f.loadByNameFn != nil {
		return f.loadByNameFn(ctx, name)
	}

	return LuaPluginDTO{}, nil
}

func (f *mockPluginLoader) LoadByNameAndVersion(ctx context.Context, name string, version string) (LuaPluginDTO, error) {
	if f.loadByNameAndVersionFn != nil {
		return f.loadByNameAndVersionFn(ctx, name, version)
	}

	return LuaPluginDTO{}, nil
}

func (f *mockPluginLoader) LoadByID(_ context.Context, _ int) (LuaPluginDTO, error) {
	return LuaPluginDTO{}, errors.New("not implemented")
}

func (f *mockPluginLoader) LoadByNameAll(_ context.Context, _ string) ([]LuaPluginDTO, error) {
	return nil, errors.New("not implemented")
}

const validPluginScript = `
plugin = {
    name = "test_plugin",
    version = "1.0.0",
    description = "test plugin",
    author = "test",

    init = function(config)
        plugin.config = config
        return true, nil
    end,

    execute = function(params)
        return {ok = true, input = params}, nil
    end,

    validate = function(params)
        return true, nil
    end,

    cleanup = function()
    end
}
`

func TestNewManager(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	repo := &mockPluginLoader{}

	manager := NewManager(cfg, repo, nil)

	require.NotNil(t, manager)
	assert.Equal(t, cfg, manager.config)
	assert.Equal(t, repo, manager.repo)
}

func TestLoadPluginsFromDBByName_WithoutVersion(t *testing.T) {
	t.Parallel()

	repo := &mockPluginLoader{
		loadByNameFn: func(_ context.Context, name string) (LuaPluginDTO, error) {
			require.Equal(t, "test_plugin", name)

			return LuaPluginDTO{PluginStr: validPluginScript}, nil
		},
	}

	manager := NewManager(&Config{}, repo, nil)

	plugin, err := manager.LoadPluginsFromDBByName(t.Context(), "test_plugin", nil)
	require.NoError(t, err)
	require.NotNil(t, plugin)

	err = plugin.Init(map[string]any{"a": "b"})
	require.NoError(t, err)
}

func TestLoadPluginsFromDBByName_WithVersion(t *testing.T) {
	t.Parallel()

	calledByName := false
	calledByVersion := false
	ver := "1.2.3"

	repo := &mockPluginLoader{
		loadByNameFn: func(_ context.Context, _ string) (LuaPluginDTO, error) {
			calledByName = true

			return LuaPluginDTO{}, nil
		},
		loadByNameAndVersionFn: func(_ context.Context, name, version string) (LuaPluginDTO, error) {
			calledByVersion = true
			require.Equal(t, "test_plugin", name)
			require.Equal(t, ver, version)

			return LuaPluginDTO{PluginStr: validPluginScript}, nil
		},
	}

	manager := NewManager(&Config{}, repo, nil)

	plugin, err := manager.LoadPluginsFromDBByName(t.Context(), "test_plugin", &ver)
	require.NoError(t, err)
	require.NotNil(t, plugin)
	assert.True(t, calledByVersion)
	assert.False(t, calledByName)
}

func TestLoadPluginsFromDBByName_RepoError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db is down")
	repo := &mockPluginLoader{
		loadByNameFn: func(_ context.Context, _ string) (LuaPluginDTO, error) {
			return LuaPluginDTO{}, repoErr
		},
	}

	manager := NewManager(&Config{}, repo, nil)

	plugin, err := manager.LoadPluginsFromDBByName(t.Context(), "test_plugin", nil)
	require.Error(t, err)
	assert.NotNil(t, plugin)
	assert.True(t, strings.Contains(err.Error(), "error loading plugin by name"))
	assert.ErrorIs(t, err, repoErr)
}

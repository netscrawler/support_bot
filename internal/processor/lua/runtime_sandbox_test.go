package lua

import (
	"testing"

	"github.com/stretchr/testify/require"
	glua "github.com/yuin/gopher-lua"
)

func TestSandboxReplacesUnexpectedGlobals(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	t.Cleanup(runtime.Close)

	runtime.vm.SetGlobal("os", glua.LString("unexpected"))
	runtime.setupSafeOS()

	_, ok := runtime.vm.GetGlobal("os").(*glua.LTable)
	require.True(t, ok)

	runtime.vm.SetGlobal("package", glua.LString("unexpected"))
	runtime.setupSafePackage()
	runtime.restoreAllowedModules()

	packageTable, ok := runtime.vm.GetGlobal("package").(*glua.LTable)
	require.True(t, ok)

	_, ok = packageTable.RawGetString("preload").(*glua.LTable)
	require.True(t, ok)
}

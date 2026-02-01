package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestNewLuaRuntime(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err, "failed to create runtime")
	require.NotNil(t, runtime)
	defer runtime.Close()

	assert.NotNil(t, runtime.GetVM())
	assert.NotNil(t, runtime.GetConfig())
}

func TestNewLuaRuntime_CustomConfig(t *testing.T) {
	t.Parallel()

	config := &RuntimeConfig{
		CallStackSize:  128,
		RegistrySize:   128,
		AllowedModules: []string{"json", "http"},
	}

	runtime, err := NewLuaRuntime(config)
	require.NoError(t, err)
	defer runtime.Close()

	assert.Equal(t, config.CallStackSize, runtime.GetConfig().CallStackSize)
}

func TestSandbox_DangerousFunctionsRemoved(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)

	defer runtime.Close()

	vm := runtime.GetVM()

	// проверяем что опасные функции удалены
	dangerousFunctions := []string{
		"dofile",
		"loadfile",
		"load",
		"module",
		"setfenv",
		"getfenv",
	}

	for _, fn := range dangerousFunctions {
		value := vm.GetGlobal(fn)
		assert.Equal(t, lua.LTNil, value.Type(),
			"dangerous function %s should be removed", fn)
	}
}

func TestSandbox_DangerousModulesRemoved(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	// проверяем что полностью опасные модули удалены
	dangerousModules := []string{
		"debug",
		"io",
	}

	for _, mod := range dangerousModules {
		value := vm.GetGlobal(mod)
		assert.Equal(t, lua.LTNil, value.Type(),
			"dangerous module %s should be removed", mod)
	}

	// проверяем что package модуль ограничен (не удален полностью, но безопасен)
	packageValue := vm.GetGlobal("package")
	assert.Equal(t, lua.LTTable, packageValue.Type(), "package should exist but be restricted")

	packageTable := packageValue.(*lua.LTable)
	// проверяем что опасные функции удалены
	assert.Equal(
		t,
		lua.LTNil,
		packageTable.RawGetString("loadlib").Type(),
		"package.loadlib should be removed",
	)
	assert.Equal(
		t,
		lua.LTNil,
		packageTable.RawGetString("searchers").Type(),
		"package.searchers should be removed",
	)

	// но preload должен быть доступен для PreloadModule
	assert.NotEqual(
		t,
		lua.LTNil,
		packageTable.RawGetString("preload").Type(),
		"package.preload should be available",
	)
}

func TestSandbox_SafeFunctionsAvailable(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	// проверяем что безопасные функции доступны
	safeFunctions := []string{
		"print",
		"type",
		"tonumber",
		"tostring",
		"pairs",
		"ipairs",
		"next",
		"pcall",
		"xpcall",
		"error",
		"assert",
	}

	for _, fn := range safeFunctions {
		value := vm.GetGlobal(fn)
		assert.NotEqual(t, lua.LTNil, value.Type(),
			"safe function %s should be available", fn)
	}
}

func TestSandbox_SafeModulesAvailable(t *testing.T) {
	t.Parallel()
	config := &RuntimeConfig{AllowedModules: []string{"io"}}

	runtime, err := NewLuaRuntime(config)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	// проверяем что безопасные модули доступны
	safeModules := []string{
		"table",
		"string",
		"math",
		"io",
	}

	for _, mod := range safeModules {
		value := vm.GetGlobal(mod)
		assert.Equal(t, lua.LTTable, value.Type(),
			"safe module %s should be available", mod)
	}
}

func TestSandbox_OSModuleRestricted(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	// проверяем что os модуль существует
	osValue := vm.GetGlobal("os")
	require.Equal(t, lua.LTTable, osValue.Type(), "os module should exist")

	osTable := osValue.(*lua.LTable)

	// проверяем что безопасные функции времени доступны
	safeFunctions := []string{"time", "clock", "date", "difftime"}
	for _, fn := range safeFunctions {
		value := osTable.RawGetString(fn)
		assert.NotEqual(t, lua.LTNil, value.Type(),
			"os.%s should be available", fn)
	}

	// проверяем что опасные функции os удалены
	dangerousFunctions := []string{"execute", "exit", "remove", "rename", "tmpname"}
	for _, fn := range dangerousFunctions {
		value := osTable.RawGetString(fn)
		assert.Equal(t, lua.LTNil, value.Type(),
			"os.%s should be removed", fn)
	}
}

func TestSandbox_ExecuteBasicLua(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	// выполняем простой Lua код
	err = vm.DoString(`
		x = 10
		y = 20
		result = x + y
	`)
	require.NoError(t, err)

	// проверяем результат
	result := vm.GetGlobal("result")
	assert.Equal(t, lua.LTNumber, result.Type())
	assert.Equal(t, lua.LNumber(30), result)
}

func TestSandbox_StringOperations(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	err = vm.DoString(`
		str = "hello world"
		upper = string.upper(str)
		length = string.len(str)
	`)
	require.NoError(t, err)

	upper := vm.GetGlobal("upper")
	assert.Equal(t, "HELLO WORLD", upper.String())

	length := vm.GetGlobal("length")
	assert.Equal(t, lua.LNumber(11), length)
}

func TestSandbox_TableOperations(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	err = vm.DoString(`
		t = {1, 2, 3, 4, 5}
		table.insert(t, 6)
		size = #t
	`)
	require.NoError(t, err)

	size := vm.GetGlobal("size")
	assert.Equal(t, lua.LNumber(6), size)
}

func TestSandbox_MathOperations(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	err = vm.DoString(`
		pi = math.pi
		sqrt = math.sqrt(16)
		max_val = math.max(10, 20, 5)
	`)
	require.NoError(t, err)

	pi := vm.GetGlobal("pi")
	assert.InDelta(t, 3.14159, float64(pi.(lua.LNumber)), 0.001)

	sqrt := vm.GetGlobal("sqrt")
	assert.Equal(t, lua.LNumber(4), sqrt)

	maxVal := vm.GetGlobal("max_val")
	assert.Equal(t, lua.LNumber(20), maxVal)
}

func TestIsModuleAllowed(t *testing.T) {
	t.Parallel()

	config := &RuntimeConfig{
		AllowedModules: []string{"json", "http", "time"},
	}

	runtime, err := NewLuaRuntime(config)
	require.NoError(t, err)
	defer runtime.Close()

	// разрешенные модули
	assert.True(t, runtime.IsModuleAllowed("json"))
	assert.True(t, runtime.IsModuleAllowed("http"))
	assert.True(t, runtime.IsModuleAllowed("time"))

	// запрещенные модули
	assert.False(t, runtime.IsModuleAllowed("os"))
	assert.False(t, runtime.IsModuleAllowed("io"))
	assert.False(t, runtime.IsModuleAllowed("debug"))
	assert.False(t, runtime.IsModuleAllowed("unknown"))
}

func TestIsModuleAllowed_EmptyWhitelist(t *testing.T) {
	t.Parallel()

	config := &RuntimeConfig{
		AllowedModules: []string{}, // пустой белый список
	}

	runtime, err := NewLuaRuntime(config)
	require.NoError(t, err)
	defer runtime.Close()

	// при пустом белом списке разрешены все модули
	assert.True(t, runtime.IsModuleAllowed("json"))
	assert.True(t, runtime.IsModuleAllowed("anything"))
}

func TestDefaultRuntimeConfig(t *testing.T) {
	t.Parallel()

	config := DefaultRuntimeConfig()

	assert.NotNil(t, config)
	assert.NotEmpty(t, config.AllowedModules)
	assert.Contains(t, config.AllowedModules, "json")
	assert.Contains(t, config.AllowedModules, "http")
	assert.Greater(t, config.CallStackSize, 0)
	assert.Greater(t, config.RegistrySize, 0)
}

func TestSandbox_PreventFileAccess(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	// пытаемся использовать dofile - должно быть nil
	err = vm.DoString(`
		if dofile ~= nil then
			error("dofile should be nil")
		end
	`)
	assert.NoError(t, err)

	// пытаемся использовать loadfile - должно быть nil
	err = vm.DoString(`
		if loadfile ~= nil then
			error("loadfile should be nil")
		end
	`)
	assert.NoError(t, err)
}

func TestSandbox_PreventDebugAccess(t *testing.T) {
	t.Parallel()

	runtime, err := NewLuaRuntime(nil)
	require.NoError(t, err)
	defer runtime.Close()

	vm := runtime.GetVM()

	// пытаемся получить доступ к debug - должно быть nil
	err = vm.DoString(`
		if debug ~= nil then
			error("debug module should be nil")
		end
	`)
	assert.NoError(t, err)
}

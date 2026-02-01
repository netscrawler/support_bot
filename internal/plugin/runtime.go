package plugins

import (
	"fmt"
	"maps"
	"slices"

	lua "github.com/yuin/gopher-lua"
)

// RuntimeConfig содержит конфигурацию для Lua runtime.
type RuntimeConfig struct {
	// AllowedModules - белый список разрешенных модулей
	AllowedModules []string

	// CallStackSize - размер стека вызовов Lua
	CallStackSize int

	// RegistrySize - размер registry для Lua
	RegistrySize int
}

// DefaultRuntimeConfig возвращает конфигурацию runtime по умолчанию.
// Это безопасные настройки для production использования.
func DefaultRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		AllowedModules: []string{
			"json",    // JSON кодирование/декодирование
			"http",    // HTTP запросы
			"url",     // парсинг URL
			"time",    // работа со временем
			"strings", // строковые операции
			"inspect", // отладка
		},
		CallStackSize: 256, // увеличен для сложных операций
		RegistrySize:  256, // стандартный размер
	}
}

// LuaRuntime представляет изолированную среду выполнения для Lua плагинов.
// Обеспечивает песочницу с ограничениями ресурсов и безопасностью.
type LuaRuntime struct {
	vm     *lua.LState    // виртуальная машина Lua
	config *RuntimeConfig // конфигурация runtime

	savedPreload *lua.LTable // сохраняем preload до очистки
}

// NewLuaRuntime создает новую изолированную среду выполнения Lua.
// Автоматически настраивает песочницу и применяет ограничения безопасности.
//
// Параметры:
//   - config: конфигурация runtime (nil = использовать по умолчанию)
//
// Возвращает:
//   - настроенный LuaRuntime с песочницей
//   - ошибку если не удалось создать VM
func NewLuaRuntime(config *RuntimeConfig) (*LuaRuntime, error) {
	if config == nil {
		config = DefaultRuntimeConfig()
	}

	// создаем Lua VM с кастомными настройками
	vm := lua.NewState(lua.Options{
		CallStackSize: config.CallStackSize,
		RegistrySize:  config.RegistrySize,
		// отключаем небезопасные функции на уровне VM
		SkipOpenLibs:        true,
		IncludeGoStackTrace: true,
	})

	runtime := &LuaRuntime{
		vm:     vm,
		config: config,
	}

	// настраиваем песочницу
	if err := runtime.CreateSandbox(); err != nil {
		vm.Close()

		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	return runtime, nil
}

// CreateSandbox настраивает безопасную песочницу для Lua кода.
// Отключает опасные модули и функции, оставляя только безопасные.
//
// Отключаемые модули и функции:
//   - os.* (кроме os.time, os.clock, os.date)
//   - io.* (файловые операции)
//   - debug.* (отладочные функции)
//   - package.loadlib (загрузка C библиотек)
//   - dofile, loadfile (загрузка файлов)
//
// Разрешаемые модули:
//   - base (без опасных функций)
//   - table, string, math
//   - разрешенные модули из конфига
func (r *LuaRuntime) CreateSandbox() error {
	// загружаем только базовые безопасные библиотеки
	r.loadSafeLibraries()

	// сохраняем preload для последующей фильтрации
	r.savePreload()

	// настраиваем os модуль (только безопасные функции)
	r.setupSafeOS()

	// настраиваем безопасный package модуль
	r.setupSafePackage()

	// удаляем опасные функции из глобального namespace
	r.removeDangerousFunctions()

	// восстанавливаем только разрешенные модули
	r.restoreAllowedModules()

	// восстанавливаем безопасный require
	r.restoreSafeRequire()

	return nil
}

// GetVM возвращает внутреннюю Lua VM.
// Используется внутри пакета для работы с виртуальной машиной.
func (r *LuaRuntime) GetVM() *lua.LState {
	return r.vm
}

// GetConfig возвращает текущую конфигурацию runtime.
func (r *LuaRuntime) GetConfig() *RuntimeConfig {
	return r.config
}

// Close корректно закрывает runtime и освобождает ресурсы.
func (r *LuaRuntime) Close() {
	if r.vm != nil {
		r.vm.Close()
	}
}

// IsModuleAllowed проверяет разрешен ли модуль в белом списке.
// Используется для контроля загрузки модулей через require().
//
// Параметры:
//   - moduleName: имя модуля для проверки
//
// Возвращает true если модуль разрешен.
func (r *LuaRuntime) IsModuleAllowed(moduleName string) bool {
	// если белый список пустой, разрешаем все
	if len(r.config.AllowedModules) == 0 {
		return true
	}

	// проверяем наличие модуля в белом списке
	return slices.Contains(r.config.AllowedModules, moduleName)
}

// setupSafeOS настраивает безопасный os модуль.
// Оставляет только функции работы со временем, удаляет execute, remove и т.д.
func (r *LuaRuntime) setupSafeOS() {
	// получаем текущий os модуль
	osValue := r.vm.GetGlobal("os")
	if osValue.Type() != lua.LTTable {
		// если os не загружен, создаем пустую таблицу
		r.vm.SetGlobal("os", r.vm.NewTable())

		return
	}

	origOS := osValue.(*lua.LTable)

	// создаем новую безопасную таблицу os
	safeOS := r.vm.NewTable()

	// копируем только безопасные функции времени
	safeFunctions := []string{"time", "clock", "date", "difftime"}
	for _, fn := range safeFunctions {
		value := origOS.RawGetString(fn)
		if value.Type() != lua.LTNil {
			safeOS.RawSetString(fn, value)
		}
	}

	// заменяем глобальный os на безопасную версию
	r.vm.SetGlobal("os", safeOS)
}

// loadSafeLibraries загружает только безопасные стандартные библиотеки Lua.
// Это базовые функции, table, string и math операции.
func (r *LuaRuntime) loadSafeLibraries() {
	// загружаем базовую библиотеку
	lua.OpenBase(r.vm)

	// загружаем безопасные модули
	lua.OpenTable(r.vm)   // table.* функции
	lua.OpenString(r.vm)  // string.* функции
	lua.OpenMath(r.vm)    // math.* функции
	lua.OpenOs(r.vm)      // os.* функции (будем фильтровать после)
	lua.OpenPackage(r.vm) // package.* для PreloadModule

	lua.OpenIo(r.vm)
	lua.OpenDebug(r.vm)
	lua.OpenCoroutine(r.vm)
}

// TODO: Добавить возможность очистки опасных функций
//
// removeDangerousFunctions удаляет опасные функции из глобального namespace.
// Эти функции могут использоваться для обхода песочницы или атак.
func (r *LuaRuntime) removeDangerousFunctions() {
	restrict := maps.Clone(RestrictedModules)

	for _, mod := range r.config.AllowedModules {
		if _, ok := restrict[mod]; ok {
			restrict[mod] = false
		}
	}

	// package и os обрабатываются отдельно, не удаляем их полностью
	delete(restrict, "package")
	delete(restrict, "os")

	// удаляем каждую опасную функцию
	for mod, allow := range restrict {
		if allow {
			r.vm.SetGlobal(mod, lua.LNil)
		}
	}
}

// setupSafePackage настраивает безопасный package модуль.
// Оставляет только preload таблицу для PreloadModule.
func (r *LuaRuntime) setupSafePackage() {
	packageValue := r.vm.GetGlobal("package")
	if packageValue.Type() != lua.LTTable {
		return
	}

	packageTable := packageValue.(*lua.LTable)
	safePackage := r.vm.NewTable()

	// копируем только preload (для PreloadModule)
	preload := packageTable.RawGetString("preload")
	if preload.Type() != lua.LTNil {
		safePackage.RawSetString("preload", preload)
	}

	// копируем loaded (для отслеживания загруженных модулей)
	loaded := packageTable.RawGetString("loaded")
	if loaded.Type() != lua.LTNil {
		safePackage.RawSetString("loaded", loaded)
	}

	r.vm.SetGlobal("package", safePackage)
}

func (r *LuaRuntime) savePreload() {
	pkg := r.vm.GetGlobal("package")
	if tbl, ok := pkg.(*lua.LTable); ok {
		preload := tbl.RawGetString("preload")
		if pl, ok := preload.(*lua.LTable); ok {
			r.savedPreload = pl
		}
	}

	if r.savedPreload == nil {
		r.savedPreload = r.vm.NewTable()
	}
}

func (r *LuaRuntime) restoreAllowedModules() {
	pkg := r.vm.GetGlobal("package")
	if pkg.Type() != lua.LTTable {
		return
	}

	// создаём новую безопасную preload таблицу
	safePreload := r.vm.NewTable()

	r.savedPreload.ForEach(func(key, val lua.LValue) {
		name := key.String()
		if r.IsModuleAllowed(name) {
			safePreload.RawSet(key, val)
		}
	})

	pkg.(*lua.LTable).RawSetString("preload", safePreload)
}

func (r *LuaRuntime) restoreSafeRequire() {
	L := r.vm

	requireFn := L.NewFunction(func(L *lua.LState) int {
		mod := L.CheckString(1)

		if !r.IsModuleAllowed(mod) {
			L.RaiseError("module '%s' is not allowed", mod)

			return 0
		}

		pkg := L.GetGlobal("package")
		pkgTbl, ok := pkg.(*lua.LTable)

		if !ok {
			L.RaiseError("package table missing")

			return 0
		}

		preload := pkgTbl.RawGetString("preload")

		plTable, ok := preload.(*lua.LTable)

		if !ok {
			L.RaiseError("preload table missing")

			return 0
		}

		loader := plTable.RawGetString(mod)
		if loader.Type() != lua.LTFunction {
			L.RaiseError("module '%s' not found", mod)

			return 0
		}

		// call loader()
		L.Push(loader)
		L.Call(0, 1)

		return 1
	})

	L.SetGlobal("require", requireFn)
}

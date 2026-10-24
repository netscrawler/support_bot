package lua

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
			"crypto",  // криптография
		},
		CallStackSize: 256, // увеличен для сложных операций
		RegistrySize:  256, // стандартный размер
	}
}

// Runtime представляет изолированную среду выполнения для Lua плагинов.
// Обеспечивает песочницу с ограничениями ресурсов и безопасностью.
type Runtime struct {
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
//   - настроенный Runtime с песочницей
//   - ошибку если не удалось создать VM
func NewLuaRuntime(config *RuntimeConfig) (*Runtime, error) {
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

	runtime := &Runtime{
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
func (r *Runtime) CreateSandbox() error {
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
func (r *Runtime) GetVM() *lua.LState {
	return r.vm
}

// GetConfig возвращает текущую конфигурацию runtime.
func (r *Runtime) GetConfig() *RuntimeConfig {
	return r.config
}

// Close корректно закрывает runtime и освобождает ресурсы.
func (r *Runtime) Close() {
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
func (r *Runtime) IsModuleAllowed(moduleName string) bool {
	// если белый список пустой, разрешаем все
	if len(r.config.AllowedModules) == 0 {
		return true
	}

	// проверяем наличие модуля в белом списке
	return slices.Contains(r.config.AllowedModules, moduleName)
}

// setupSafeOS настраивает безопасный os модуль.
// Оставляет только функции работы со временем, удаляет execute, remove и т.д.
func (r *Runtime) setupSafeOS() {
	// получаем текущий os модуль
	osValue := r.vm.GetGlobal("os")
	origOS, ok := osValue.(*lua.LTable)

	if !ok {
		// если os не загружен, создаем пустую таблицу
		r.vm.SetGlobal("os", r.vm.NewTable())

		return
	}

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
func (r *Runtime) loadSafeLibraries() {
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
func (r *Runtime) removeDangerousFunctions() {
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
func (r *Runtime) setupSafePackage() {
	packageValue := r.vm.GetGlobal("package")
	packageTable, ok := packageValue.(*lua.LTable)

	if !ok {
		r.vm.SetGlobal("package", r.vm.NewTable())

		return
	}

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

func (r *Runtime) savePreload() {
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

func (r *Runtime) restoreAllowedModules() {
	pkg, ok := r.vm.GetGlobal("package").(*lua.LTable)
	if !ok {
		pkg = r.vm.NewTable()
		r.vm.SetGlobal("package", pkg)
	}

	// создаём новую безопасную preload таблицу
	safePreload := r.vm.NewTable()

	r.savedPreload.ForEach(func(key, val lua.LValue) {
		name := key.String()
		if r.IsModuleAllowed(name) {
			safePreload.RawSet(key, val)
		}
	})

	pkg.RawSetString("preload", safePreload)
}

func (r *Runtime) restoreSafeRequire() {
	l := r.vm

	requireFn := l.NewFunction(func(l *lua.LState) int {
		mod := l.CheckString(1)

		if !r.IsModuleAllowed(mod) {
			l.RaiseError("module '%s' is not allowed", mod)

			return 0
		}

		pkg := l.GetGlobal("package")
		pkgTbl, ok := pkg.(*lua.LTable)

		if !ok {
			l.RaiseError("package table missing")

			return 0
		}

		preload := pkgTbl.RawGetString("preload")

		plTable, ok := preload.(*lua.LTable)

		if !ok {
			l.RaiseError("preload table missing")

			return 0
		}

		loader := plTable.RawGetString(mod)
		if loader.Type() != lua.LTFunction {
			l.RaiseError("module '%s' not found", mod)

			return 0
		}

		// call loader()
		l.Push(loader)
		l.Call(0, 1)

		return 1
	})

	l.SetGlobal("require", requireFn)
}

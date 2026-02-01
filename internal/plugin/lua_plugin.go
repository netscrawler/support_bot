package plugins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"support_bot/internal/plugin/stdlib"

	"github.com/cjoudrey/gluahttp"
	"github.com/cjoudrey/gluaurl"
	json "github.com/layeh/gopher-json"
	"github.com/tengattack/gluacrypto"
	libs "github.com/vadv/gopher-lua-libs"
	lua "github.com/yuin/gopher-lua"
)

// LuaPlugin - обертка вокруг Lua-скрипта, реализующая интерфейс Plugin.
// Каждый экземпляр содержит отдельную Lua VM для изоляции выполнения.
type LuaPlugin struct {
	MetaData

	filePath string // путь к файлу .lua на диске

	// Runtime и статистика
	runtime *LuaRuntime  // безопасная среда выполнения с песочницей
	vm      *lua.LState  // виртуальная машина Lua (прямая ссылка из runtime)
	stdlib  *stdlib.STD  // стандартная библиотека для плагинов
	mu      sync.RWMutex // мьютекс для потокобезопасного доступа
}

// NewLuaPlugin создает новый экземпляр Lua-плагина из файла.
// Загружает Lua-скрипт в безопасной песочнице, инициализирует VM и извлекает метаданные.
//
// Параметры:
//   - filePath: путь к .lua файлу плагина
//
// Возвращает:
//   - указатель на LuaPlugin при успехе
//   - ошибку если файл не найден, содержит ошибки или некорректные метаданные
func NewLuaPlugin(filePath string) (*LuaPlugin, error) {
	return NewLuaPluginWithConfig(filePath, DefaultRuntimeConfig(), nil)
}

// NewLuaPluginWithConfig создает плагин с кастомной конфигурацией runtime.
// Позволяет настроить лимиты памяти, таймауты и белый список модулей.
//
// Параметры:
//   - filePath: путь к .lua файлу плагина
//   - config: конфигурация runtime (nil = использовать по умолчанию)
//   - std: стандартная библиотека для плагинов (nil = без stdlib)
//
// Возвращает настроенный плагин в безопасной песочнице.
func NewLuaPluginWithConfig(
	filePath string,
	config *RuntimeConfig,
	std *stdlib.STD,
) (*LuaPlugin, error) {
	// создаем безопасную среду выполнения с песочницей
	runtime, err := NewLuaRuntime(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime: %w", err)
	}

	vm := runtime.GetVM()

	plugin := &LuaPlugin{
		filePath: filePath,
		runtime:  runtime,
		vm:       vm,
		stdlib:   std,
	}

	// предзагружаем дополнительные модули для плагинов
	// базовые (table, string, math, os) уже загружены в песочнице
	plugin.preloadModules()

	// загружаем и выполняем Lua-скрипт
	// при этом определяются все функции и глобальная таблица plugin
	if err := vm.DoFile(filePath); err != nil {
		runtime.Close()

		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// извлекаем метаданные из глобальной таблицы plugin
	if err := plugin.loadMetadata(); err != nil {
		runtime.Close()

		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	return plugin, nil
}

// Init инициализирует плагин с заданной конфигурацией.
// Вызывает Lua-функцию plugin.init(config) и передает параметры конфигурации.
//
// Параметры:
//   - config: карта с параметрами конфигурации из YAML (api_key, url и т.д.)
//
// Lua-функция должна вернуть два значения: (success bool, error string/nil)
// Возвращает ошибку если инициализация не удалась.
func (p *LuaPlugin) Init(config map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// конвертируем Go map в Lua table используя gopher-json
	// это автоматически обрабатывает вложенные структуры
	configValue := json.DecodeValue(p.vm, config)

	// вызываем Lua-функцию plugin.init(config)
	// NRet:2 означает что ожидаем 2 возвращаемых значения
	// Protect:true включает защиту от паники в Lua-коде
	if err := p.vm.CallByParam(lua.P{
		//nolint:errcheck // проверка типа выше
		Fn:      p.vm.GetGlobal("plugin").(*lua.LTable).RawGetString("init"),
		NRet:    2,
		Protect: true,
	}, configValue); err != nil {
		return fmt.Errorf("plugin init failed: %w", err)
	}

	// читаем возвращаемые значения: success, error
	// Get(-2) - предпоследнее значение (success)
	// Get(-1) - последнее значение (error)
	success := p.vm.Get(-2)
	errValue := p.vm.Get(-1)
	p.vm.Pop(2) // очищаем стек

	// если success == true, инициализация прошла успешно
	if lua.LVAsBool(success) {
		return nil
	}

	// если вернулась ошибка, возвращаем её
	if errValue.Type() != lua.LTNil {
		return fmt.Errorf("plugin init error: %s", errValue.String())
	}

	return ErrPluginInitFailed
}

// Execute запрашивает данные из внешнего источника через плагин.
// Это основная рабочая функция, которая вызывает plugin.execute(params) в Lua.
//
// Параметры:
//   - ctx: контекст для отмены операции (пока не используется, будет в этапе 2)
//   - params: параметры запроса (URL, фильтры, ID карточки Metabase и т.д.)
//
// Возвращает:
//   - []byte: данные в формате JSON
//   - error: ошибка если запрос не удался
//
// Функция автоматически собирает статистику вызовов для мониторинга.
func (p *LuaPlugin) Execute(_ context.Context, params map[string]any) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// конвертируем параметры из Go map в Lua table
	paramsValue := json.DecodeValue(p.vm, params)

	fn, ok := p.vm.GetGlobal("plugin").(*lua.LTable)
	if !ok {
		return nil, errors.New("not found global plugin")
	}

	// вызываем Lua-функцию plugin.fetch_data(params)
	// она должна вернуть: (data, error)
	paramsP := lua.P{
		Fn:      fn.RawGetString("execute"),
		NRet:    2,
		Protect: true,
	}

	if err := p.vm.CallByParam(paramsP, paramsValue); err != nil {
		return nil, fmt.Errorf("plugin execute failed: %w", err)
	}

	// читаем возвращаемые значения со стека
	dataValue := p.vm.Get(-2) // данные (table, string или другой тип)
	errValue := p.vm.Get(-1)  // ошибка (string или nil)
	p.vm.Pop(2)               // очищаем стек

	// если плагин вернул ошибку, обрабатываем её
	if errValue.Type() != lua.LTNil {
		err := fmt.Errorf("plugin error: %s", errValue.String())

		return nil, err
	}

	// конвертируем результат (Lua value) в JSON bytes
	// gopher-json автоматически обрабатывает tables, strings, numbers и т.д.
	data, err := json.Encode(dataValue)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to JSON: %w", err)
	}

	return data, nil
}

// Validate проверяет корректность параметров перед выполнением Execute.
// Вызывает Lua-функцию plugin.validate(params) для проверки обязательных полей.
//
// Параметры:
//   - params: параметры для проверки (те же что передаются в Execute)
//
// Возвращает:
//   - nil если параметры корректны
//   - error с описанием проблемы если валидация не прошла
//
// Это позволяет отловить ошибки конфигурации до начала выполнения запроса.
func (p *LuaPlugin) Validate(params map[string]any) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// конвертируем параметры для передачи в Lua
	paramsValue := json.DecodeValue(p.vm, params)

	// получаем таблицу plugin и проверяем её тип
	pluginTable := p.vm.GetGlobal("plugin")
	if pluginTable.Type() != lua.LTTable {
		return ErrPluginTableNotFound
	}

	// вызываем Lua-функцию plugin.validate(params)
	// она должна вернуть: (valid bool, error string/nil)
	validateFunc := pluginTable.(*lua.LTable).RawGetString(
		"validate",
	)

	if err := p.vm.CallByParam(lua.P{
		Fn:      validateFunc,
		NRet:    2,
		Protect: true,
	}, paramsValue); err != nil {
		return fmt.Errorf("plugin validate failed: %w", err)
	}

	// читаем результат валидации
	valid := p.vm.Get(-2)    // bool: true если валидация прошла
	errValue := p.vm.Get(-1) // string/nil: сообщение об ошибке
	p.vm.Pop(2)

	// если valid == true, параметры корректны
	if lua.LVAsBool(valid) {
		return nil
	}

	// возвращаем ошибку валидации
	if errValue.Type() != lua.LTNil {
		return fmt.Errorf("validation error: %s", errValue.String())
	}

	return ErrValidationFailed
}

// Cleanup освобождает ресурсы плагина при выгрузке.
// Вызывает Lua-функцию plugin.cleanup() для закрытия соединений,
// освобождения памяти и других ресурсов, а затем закрывает Lua VM.
//
// Должна вызываться при:
//   - остановке приложения
//   - перезагрузке плагина
//   - выгрузке плагина из системы
//
// После вызова Cleanup плагин становится неработоспособным.
func (p *LuaPlugin) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// получаем таблицу plugin и проверяем её тип
	pluginTable := p.vm.GetGlobal("plugin")
	if pluginTable.Type() != lua.LTTable {
		p.runtime.Close()

		return ErrPluginTableNotFound
	}

	// вызываем Lua-функцию plugin.cleanup()
	// плагин может закрыть соединения, сохранить состояние и т.д.
	cleanupFunc := pluginTable.(*lua.LTable).RawGetString(
		"cleanup",
	)

	if err := p.vm.CallByParam(lua.P{
		Fn:      cleanupFunc,
		NRet:    0, // не ожидаем возвращаемых значений
		Protect: true,
	}); err != nil {
		p.runtime.Close()

		return fmt.Errorf("plugin cleanup failed: %w", err)
	}

	// закрываем Lua VM и освобождаем память
	p.runtime.Close()

	return nil
}

// IsHealthy проверяет работоспособность плагина.
// Используется для health checks и мониторинга.
//
// Плагин считается неработоспособным если:
//   - последний вызов завершился ошибкой
//   - ошибка произошла менее минуты назад
//
// Это позволяет отслеживать проблемные плагины в реальном времени.
func (p *LuaPlugin) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// FIX: переделать систему определения здоровья у плагина.
	// если последняя ошибка была недавно (< 1 минуты), плагин нездоров
	// if p.stats.LastError != nil {
	// 	if time.Since(p.stats.LastCallTime) < time.Minute {
	// 		return false
	// 	}
	// }

	return true
}

// loadMetadata извлекает метаданные плагина из глобальной Lua-таблицы plugin.
// Ожидается что в Lua-скрипте определена таблица plugin с полями:
//   - name (обязательно): уникальное имя плагина
//   - version: версия плагина
//   - description: описание функциональности
//   - author: автор плагина
//
// Возвращает ошибку если таблица plugin не найдена или name пустое.
func (p *LuaPlugin) loadMetadata() error {
	// получаем глобальную переменную plugin
	pluginTable := p.vm.GetGlobal("plugin")
	if pluginTable.Type() != lua.LTTable {
		return ErrPluginTableNotFound
	}

	table, ok := pluginTable.(*lua.LTable)
	if !ok {
		return ErrPluginTableNotFound
	}

	// извлекаем все поля метаданных из таблицы
	p.name = table.RawGetString("name").String()
	p.version = table.RawGetString("version").String()
	p.description = table.RawGetString("description").String()
	p.author = table.RawGetString("author").String()

	// имя плагина обязательно - оно используется как уникальный идентификатор
	if p.name == "" {
		return ErrPluginNameRequired
	}

	return nil
}

// preloadModules загружает дополнительные модули в песочницу.
// Это модули которые нужны для работы с внешними API и данными.
func (p *LuaPlugin) preloadModules() {
	pkg := p.vm.GetGlobal("package")
	if pkg == lua.LNil {
		tbl := p.vm.NewTable()
		p.vm.SetGlobal("package", tbl)
		p.vm.SetField(tbl, "preload", p.vm.NewTable())
	}

	// JSON - работа с JSON данными (encode/decode)
	p.vm.PreloadModule("json", json.Loader)

	// HTTP - выполнение HTTP запросов (GET, POST, PUT, DELETE)
	// создаем HTTP клиент с разумными таймаутами
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	p.vm.PreloadModule("http", gluahttp.NewHttpModule(httpClient).Loader)

	// URL - парсинг и построение URL
	p.vm.PreloadModule("url", gluaurl.Loader)

	// Crypto - криптографические функции (md5, sha256, hmac и т.д.)
	gluacrypto.Preload(p.vm)

	// Time, inspect, strings и другие утилиты из gopher-lua-libs
	// это добавляет много полезных функций для работы со временем, строками и т.д.
	libs.Preload(p.vm)

	// Регистрируем stdlib если он передан
	if p.stdlib != nil {
		p.stdlib.Register(p.vm)
	}
}

type MetaData struct {
	// Метаданные плагина, извлекаемые из Lua-таблицы plugin
	name        string // уникальное имя плагина
	version     string // версия в формате semver
	description string // описание функциональности
	author      string // автор плагина
}

// Name возвращает уникальное имя плагина.
// Используется как идентификатор при регистрации в Manager.
func (p *MetaData) Name() string {
	return p.name
}

// Version возвращает версию плагина в формате semver.
// Например: "1.0.0", "2.1.3-beta".
func (p *MetaData) Version() string {
	return p.version
}

// Description возвращает человекочитаемое описание плагина.
// Используется в UI и логах для понимания назначения плагина.
func (p *MetaData) Description() string {
	return p.description
}

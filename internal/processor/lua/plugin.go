package lua

import (
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"net/http"
	"support_bot/internal/models"
	"sync"
	"time"

	luapkg "support_bot/internal/pkg/lua"
	"support_bot/internal/processor/lua/stdlib"

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
	pluginStr string

	// Runtime и статистика
	runtime *LuaRuntime  // безопасная среда выполнения с песочницей
	vm      *lua.LState  // виртуальная машина Lua (прямая ссылка из runtime)
	stdlib  *stdlib.STD  // стандартная библиотека для плагинов
	mu      sync.RWMutex // мьютекс для потокобезопасного доступа
}

// NewLuaPlugin создает плагин с кастомной конфигурацией runtime.
// Позволяет настроить лимиты памяти, таймауты и белый список модулей.
//
// Параметры:
//   - filePath: путь к .lua файлу плагина
//   - config: конфигурация runtime (nil = использовать по умолчанию)
//   - std: стандартная библиотека для плагинов (nil = без stdlib)
//
// Возвращает настроенный плагин в безопасной песочнице.
func NewLuaPlugin(
	pluginStr string,
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
		pluginStr: pluginStr,
		runtime:   runtime,
		vm:        vm,
		stdlib:    std,
	}

	// предзагружаем дополнительные модули для плагинов
	// базовые (table, string, math, os) уже загружены в песочнице
	plugin.preloadModules()

	if err := vm.DoString(pluginStr); err != nil {
		vm.Close()

		return nil, fmt.Errorf(
			"failed to load lua plugin: %w",
			err,
		)
	}
	process := vm.GetGlobal("process")

	if process.Type() != lua.LTFunction {
		return nil, errors.New(
			"lua plugin must define process(input)",
		)
	}

	return plugin, nil
}

// Execute запрашивает данные из внешнего источника через плагин.
// Это основная рабочая функция, которая вызывает plugin.process(params) в Lua.
//
// Параметры:
//   - ctx: контекст для отмены операции (пока не используется, будет в этапе 2)
//   - input: map[string][]map[string]any - Данные для обработки
//
// Возвращает:
//   - map[string][]map[string]any - обработанные данные
//   - error: ошибка если запрос не удался
//
// Функция автоматически собирает статистику вызовов для мониторинга.
func (p *LuaPlugin) Execute(
	ctx context.Context,
	input models.Dataset,
) (models.Dataset, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	process := p.vm.GetGlobal("process")

	if process.Type() != lua.LTFunction {
		return nil, errors.New(
			"lua function process(input) not found",
		)
	}

	inputValue := luapkg.MapResultToLua(
		p.vm,
		input,
	)

	errValue := p.vm.CallByParam(
		lua.P{
			Fn:      process,
			NRet:    2,
			Protect: true,
		},
		inputValue,
	)

	if errValue != nil {
		return nil, fmt.Errorf(
			"plugin.process failed: %w",
			errValue,
		)
	}

	result := p.vm.Get(-2)
	luaErr := p.vm.Get(-1)

	p.vm.Pop(2)

	if luaErr.Type() != lua.LTNil {
		return nil, fmt.Errorf(
			"plugin error: %s",
			luaErr.String(),
		)
	}

	data, ok := luaToMapResult(result)
	if !ok {
		return nil, errors.New(
			"plugin.process must return map[string][]map[string]any",
		)
	}

	return data, nil
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

	// URL - парсинг и построение URL
	p.vm.PreloadModule("url", gluaurl.Loader)

	// Crypto - криптографические функции (md5, sha256, hmac и т.д.)
	gluacrypto.Preload(p.vm)

	// Time, inspect, strings и другие утилиты из gopher-lua-libs
	// это добавляет много полезных функций для работы со временем, строками и т.д.
	libs.Preload(p.vm)

	// HTTP - выполнение HTTP запросов (GET, POST, PUT, DELETE)
	// создаем HTTP клиент с разумными таймаутами
	// важно: preload после libs.Preload, чтобы переопределить модуль "http"
	// из gopher-lua-libs и гарантировать API gluahttp.
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	p.vm.PreloadModule("http", gluahttp.NewHttpModule(httpClient).Loader)

	// Регистрируем stdlib если он передан
	if p.stdlib != nil {
		p.stdlib.Register(p.vm)
	}
}

func normalizeForLua(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	data, err := stdjson.Marshal(value)
	if err != nil {
		return nil, err
	}

	var normalized any
	if err := stdjson.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func luaToMapResult(
	value lua.LValue,
) (map[string][]map[string]any, bool) {
	root, ok := value.(*lua.LTable)
	if !ok {
		return nil, false
	}

	result := make(
		map[string][]map[string]any,
	)

	root.ForEach(func(k, v lua.LValue) {
		name := k.String()

		arr, ok := v.(*lua.LTable)
		if !ok {
			return
		}

		rows := make(
			[]map[string]any,
			0,
			arr.Len(),
		)

		arr.ForEach(func(_, rowValue lua.LValue) {
			rowTable, ok := rowValue.(*lua.LTable)
			if !ok {
				return
			}

			row := make(
				map[string]any,
			)

			rowTable.ForEach(func(k, v lua.LValue) {
				row[k.String()] = luapkg.LuaValueToGo(v)
			})

			rows = append(rows, row)
		})

		result[name] = rows
	})

	return result, true
}

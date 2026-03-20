// Package plugins предоставляет систему расширяемых Lua-плагинов для динамического
// добавления функциональности без перекомпиляции приложения.
//
// # Обзор
//
// Пакет plugins реализует безопасную систему выполнения Lua-скриптов в изолированной
// песочнице с контролем доступа к ресурсам. Плагины используются для:
//   - Интеграции с внешними API и источниками данных
//   - Обработки и трансформации данных
//   - Расширения бизнес-логики без изменения кода приложения
//   - Создания пользовательских коллекторов и обработчиков
//
// # Архитектура
//
// Система плагинов состоит из следующих компонентов:
//
//	┌──────────────────────────────────────────────────┐
//	│              Manager (менеджер)                   │
//	│  - Загрузка плагинов из БД                       │
//	│  - Управление жизненным циклом                   │
//	│  - Версионирование                               │
//	└────────────┬─────────────────────────────────────┘
//	             │
//	             ▼
//	┌──────────────────────────────────────────────────┐
//	│           LuaPlugin (реализация)                 │
//	│  - Интерфейс Plugin                              │
//	│  - Выполнение Lua кода                           │
//	│  - Сериализация/десериализация данных            │
//	└────────────┬─────────────────────────────────────┘
//	             │
//	             ▼
//	┌──────────────────────────────────────────────────┐
//	│          LuaRuntime (песочница)                  │
//	│  - Изоляция выполнения                           │
//	│  - Контроль доступа к модулям                    │
//	│  - Ограничение опасных операций                  │
//	└──────────────────────────────────────────────────┘
//
// # Безопасность
//
// Система плагинов реализует многоуровневую модель безопасности:
//
//  1. Изоляция выполнения - каждый плагин выполняется в отдельной Lua VM
//  2. Песочница (sandbox) - удаление опасных функций и модулей:
//     - Отключены: io.*, debug.*, os.execute, dofile, loadfile
//     - Разрешены: table, string, math, os.time, json, http (через белый список)
//  3. Белый список модулей - явное указание разрешенных внешних библиотек
//  4. Контроль ресурсов - ограничение размера стека и registry
//
// # Использование
//
// ## Создание и загрузка плагина из БД
//
//	// Инициализация менеджера
//	cfg := &plugins.Config{
//		Enable: true,
//		AllowedModules: []string{"json", "http", "url"},
//	}
//	repo := &PluginRepository{db: db}
//	manager := plugins.NewManager(cfg, repo, stdlib)
//
//	// Загрузка последней версии плагина
//	plugin, err := manager.NewPlugin(ctx, "metabase_collector", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Загрузка конкретной версии
//	version := "1.2.3"
//	plugin, err := manager.NewPlugin(ctx, "metabase_collector", &version)
//
// ## Создание плагина из строки
//
//	script := `
//	plugin = {
//	    name = "example",
//	    version = "1.0.0",
//	    description = "Пример плагина",
//	    author = "Developer",
//
//	    init = function(config)
//	        plugin.api_key = config.api_key
//	        return true, nil
//	    end,
//
//	    execute = function(params)
//	        local http = require("http")
//	        local json = require("json")
//
//	        local response, err = http.request("GET", params.url)
//	        if err then
//	            return nil, err
//	        end
//
//	        local data = json.decode(response.body)
//	        return {result = data}, nil
//	    end,
//
//	    validate = function(params)
//	        if not params.url then
//	            return false, "url is required"
//	        end
//	        return true, nil
//	    end,
//
//	    cleanup = function()
//	        plugin.api_key = nil
//	    end
//	}
//	`
//
//	plugin, err := plugins.NewLuaPluginWithConfigFromString(
//	    script,
//	    plugins.DefaultRuntimeConfig(),
//	    stdlib,
//	)
//
// ## Работа с плагином
//
//	// Инициализация с конфигурацией
//	config := map[string]any{
//	    "api_key": "secret",
//	    "base_url": "https://api.example.com",
//	}
//	if err := plugin.Init(config); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Валидация параметров
//	params := map[string]any{
//	    "url": "https://api.example.com/data",
//	    "filter": "active",
//	}
//	if err := plugin.Validate(params); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Выполнение
//	data, err := plugin.Execute(ctx, params)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Обработка результата (JSON)
//	var result map[string]any
//	json.Unmarshal(data, &result)
//
//	// Освобождение ресурсов
//	defer plugin.Cleanup()
//
// # Структура Lua плагина
//
// Каждый плагин должен определять глобальную таблицу `plugin` с метаданными
// и обязательными функциями:
//
//	plugin = {
//	    -- Метаданные (обязательно)
//	    name = "plugin_name",           -- Уникальное имя
//	    version = "1.0.0",              -- Semver версия
//	    description = "Description",    -- Описание
//	    author = "Author Name",         -- Автор
//
//	    -- Инициализация (обязательно)
//	    -- Вызывается один раз при загрузке
//	    -- Возвращает: (success: bool, error: string|nil)
//	    init = function(config)
//	        -- Сохранение конфигурации
//	        plugin.config = config
//	        return true, nil
//	    end,
//
//	    -- Основная функция выполнения (обязательно)
//	    -- Возвращает: (data: table|string, error: string|nil)
//	    execute = function(params)
//	        -- Бизнес-логика
//	        return {result = "ok"}, nil
//	    end,
//
//	    -- Валидация параметров (обязательно)
//	    -- Возвращает: (valid: bool, error: string|nil)
//	    validate = function(params)
//	        if not params.required_field then
//	            return false, "required_field is missing"
//	        end
//	        return true, nil
//	    end,
//
//	    -- Очистка ресурсов (обязательно)
//	    cleanup = function()
//	        plugin.config = nil
//	    end
//	}
//
// # Доступные модули в плагинах
//
// По умолчанию плагинам доступны следующие модули:
//
//   - json - кодирование/декодирование JSON (github.com/layeh/gopher-json)
//   - http - HTTP клиент (github.com/cjoudrey/gluahttp)
//   - url - парсинг и построение URL (github.com/cjoudrey/gluaurl)
//   - crypto - криптографические функции: md5, sha256, hmac (github.com/tengattack/gluacrypto)
//   - time - работа со временем (github.com/vadv/gopher-lua-libs)
//   - strings - дополнительные строковые операции
//   - inspect - отладка и инспекция данных
//
// Стандартные модули Lua (всегда доступны):
//   - table - операции с таблицами
//   - string - работа со строками
//   - math - математические функции
//   - os.time, os.clock, os.date - функции времени (остальные os.* отключены)
//
// # Примеры плагинов
//
// ## HTTP коллектор данных
//
//	plugin = {
//	    name = "http_collector",
//	    version = "1.0.0",
//	    description = "Сбор данных через HTTP API",
//	    author = "Team",
//
//	    init = function(config)
//	        plugin.base_url = config.base_url
//	        plugin.api_key = config.api_key
//	        return true, nil
//	    end,
//
//	    execute = function(params)
//	        local http = require("http")
//	        local json = require("json")
//
//	        local url = plugin.base_url .. params.endpoint
//	        local headers = {
//	            ["Authorization"] = "Bearer " .. plugin.api_key,
//	            ["Content-Type"] = "application/json"
//	        }
//
//	        local response, err = http.request("GET", url, {headers = headers})
//	        if err then
//	            return nil, "HTTP request failed: " .. err
//	        end
//
//	        if response.status_code ~= 200 then
//	            return nil, "Bad status: " .. response.status_code
//	        end
//
//	        local data = json.decode(response.body)
//	        return data, nil
//	    end,
//
//	    validate = function(params)
//	        if not params.endpoint then
//	            return false, "endpoint is required"
//	        end
//	        return true, nil
//	    end,
//
//	    cleanup = function()
//	        plugin.api_key = nil
//	    end
//	}
//
// ## Трансформация данных
//
//	plugin = {
//	    name = "data_enricher",
//	    version = "1.0.0",
//	    description = "Обогащение данных дополнительными полями",
//	    author = "Team",
//
//	    init = function(config)
//	        return true, nil
//	    end,
//
//	    execute = function(params)
//	        local result = {}
//
//	        for key, items in pairs(params.data) do
//	            result[key] = {}
//	            for i, item in ipairs(items) do
//	                local enriched = {
//	                    id = item.id,
//	                    name = item.name,
//	                    -- Добавляем новые поля
//	                    processed = true,
//	                    processed_at = os.time(),
//	                    index = i
//	                }
//	                table.insert(result[key], enriched)
//	            end
//	        end
//
//	        return result, nil
//	    end,
//
//	    validate = function(params)
//	        if not params.data then
//	            return false, "data is required"
//	        end
//	        return true, nil
//	    end,
//
//	    cleanup = function() end
//	}
//
// # API интерфейсы
//
// ## Plugin interface
//
// Основной интерфейс для работы с плагинами:
//
//	type Plugin interface {
//	    // Метаданные
//	    Name() string
//	    Version() string
//	    Description() string
//
//	    // Жизненный цикл
//	    Init(config map[string]any) error
//	    Execute(ctx context.Context, params map[string]any) ([]byte, error)
//	    Validate(params map[string]any) error
//	    Cleanup() error
//
//	    // Мониторинг
//	    IsHealthy() bool
//	}
//
// ## PluginLoader interface
//
// Интерфейс для загрузки плагинов из хранилища:
//
//	type PluginLoader interface {
//	    LoadByName(ctx context.Context, name string) (LuaPluginDTO, error)
//	    LoadByNameAndVersion(ctx context.Context, name, version string) (LuaPluginDTO, error)
//	    LoadByID(ctx context.Context, id int) (LuaPluginDTO, error)
//	    LoadByNameAll(ctx context.Context, name string) ([]LuaPluginDTO, error)
//	}
//
// ## Manager API
//
// Менеджер предоставляет высокоуровневый API для работы с плагинами:
//
//	// Создание менеджера
//	func NewManager(cfg *Config, repo PluginLoader, std *stdlib.STD) *Manager
//
//	// Загрузка плагина из БД
//	func (m *Manager) NewPlugin(ctx context.Context, name string, version *string) (*LuaPlugin, error)
//
//	// Построение плагина из DTO
//	func (m *Manager) BuildPluginFromDTO(plug LuaPluginDTO) (*LuaPlugin, error)
//
// # Конфигурация
//
// Система плагинов настраивается через структуру Config:
//
//	type Config struct {
//	    Enable           bool          // Включить/выключить систему плагинов
//	    PluginsDir       string        // Директория с Lua файлами (deprecated)
//	    LoadTimeout      time.Duration // Таймаут загрузки плагина
//	    ExecutionTimeout time.Duration // Таймаут выполнения плагина
//	    MaxMemoryMB      int           // Лимит памяти на плагин (МБ)
//	    AllowedModules   []string      // Белый список модулей
//	}
//
// Пример конфигурации:
//
//	cfg := &plugins.Config{
//	    Enable:           true,
//	    LoadTimeout:      5 * time.Second,
//	    ExecutionTimeout: 30 * time.Second,
//	    MaxMemoryMB:      128,
//	    AllowedModules:   []string{"json", "http", "url", "crypto"},
//	}
//
// # Обработка ошибок
//
// Пакет определяет специфичные типы ошибок:
//
//	var (
//	    ErrPluginTableNotFound   = errors.New("plugin table not found")
//	    ErrPluginNameRequired    = errors.New("plugin name is required")
//	    ErrPluginInitFailed      = errors.New("plugin init failed")
//	    ErrValidationFailed      = errors.New("validation failed")
//	    ErrPluginManagerDisabled = errors.New("plugin manager disabled")
//	)
//
// Проверка ошибок:
//
//	if err := plugin.Init(config); err != nil {
//	    if errors.Is(err, plugins.ErrPluginInitFailed) {
//	        // Обработка ошибки инициализации
//	    }
//	}
//
// # Best Practices
//
// 1. Всегда вызывайте Cleanup() при завершении работы с плагином
// 2. Используйте контексты с таймаутом при вызове Execute()
// 3. Валидируйте параметры перед выполнением
// 4. Обрабатывайте ошибки Lua-кода gracefully
// 5. Ограничивайте доступные модули минимально необходимым набором
// 6. Храните плагины в БД для версионирования и аудита
// 7. Тестируйте плагины в изолированном окружении перед продакшеном
//
// # Ограничения
//
//   - Нельзя использовать io операции (чтение/запись файлов)
//   - Нельзя выполнять системные команды (os.execute)
//   - Нельзя загружать произвольные C библиотеки
//   - Нельзя использовать debug модуль
//   - Ограничения на размер стека и registry
//
// # См. также
//
//   - github.com/yuin/gopher-lua - Lua VM для Go
//   - github.com/layeh/gopher-json - JSON модуль для Lua
//   - github.com/cjoudrey/gluahttp - HTTP клиент для Lua
//
// # Версия
//
// Текущая версия: 2.0.0 (март 2026)
// - Добавлена загрузка из БД с версионированием
// - Улучшена система безопасности песочницы
// - Расширен набор доступных модулей
// - Добавлена поддержка stdlib
package plugins

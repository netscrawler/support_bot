package plugins

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
)

// Manager управляет жизненным циклом всех загруженных плагинов.
// Обеспечивает потокобезопасную загрузку, выгрузку и доступ к плагинам.
//
// Основные функции:
//   - Загрузка плагинов из директории
//   - Регистрация и хранение плагинов по имени
//   - Предоставление доступа к плагинам
//   - Перезагрузка плагинов без остановки системы
//   - Graceful shutdown всех плагинов
type Manager struct {
	plugins map[string]Plugin // карта загруженных плагинов (ключ - имя плагина)
	config  *Config           // конфигурация системы плагинов
	mu      sync.RWMutex      // мьютекс для потокобезопасного доступа
}

// NewManager создает новый менеджер плагинов с заданной конфигурацией.
//
// Параметры:
//   - cfg: конфигурация плагинов (enable, plugins_dir, таймауты и т.д.)
//
// Возвращает готовый к использованию Manager с пустой картой плагинов.
func NewManager(cfg *Config) (*Manager, error) {
	m := &Manager{
		plugins: make(map[string]Plugin),
		config:  cfg,
	}

	// Автоматически загружает плагины из директории.
	err := m.autoLoad()
	if err != nil {
		err = fmt.Errorf("auto load plugins from %s failed : %w", m.config.PluginsDir, err)
	}

	return m, err
}

// LoadPlugin загружает один плагин из указанного файла.
// Создает новый экземпляр LuaPlugin и регистрирует его в системе.
//
// Параметры:
//   - filePath: полный путь к .lua файлу плагина
//
// Возвращает ошибку если:
//   - файл не найден или содержит ошибки
//   - плагин с таким именем уже загружен
//   - не удалось извлечь метаданные
//
// После успешной загрузки плагин доступен через GetPlugin(name).
func (m *Manager) LoadPlugin(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enable {
		return ErrPluginManagerDisabled
	}

	// создаем конфигурацию runtime из AllowedModules
	runtimeCfg := &RuntimeConfig{
		AllowedModules: m.config.AllowedModules,
		CallStackSize:  256,
		RegistrySize:   256,
	}

	// создаем новый экземпляр Lua-плагина
	// при этом загружается и выполняется Lua-скрипт
	plugin, err := NewLuaPluginWithConfig(filePath, runtimeCfg, m.config.Stdlib)
	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}

	// проверяем что плагин с таким именем еще не загружен
	// имена плагинов должны быть уникальными
	if _, exists := m.plugins[plugin.Name()]; exists {
		_ = plugin.Cleanup() // освобождаем ресурсы нового плагина

		return fmt.Errorf("plugin %s already loaded", plugin.Name())
	}

	// регистрируем плагин в карте
	m.plugins[plugin.Name()] = plugin

	return nil
}

// LoadPluginsFromDir загружает все .lua файлы из указанной директории.
// Сканирует директорию и загружает каждый найденный .lua файл как плагин.
//
// Параметры:
//   - dir: путь к директории с плагинами
//
// Возвращает ошибку если:
//   - не удалось прочитать директорию
//   - хотя бы один плагин не загрузился
//
// Если config.Enable == false, функция ничего не делает и возвращает nil.
// Это позволяет отключить систему плагинов через конфигурацию.
func (m *Manager) LoadPluginsFromDir(dir string) error {
	// если система плагинов отключена, ничего не делаем
	if !m.config.Enable {
		return ErrPluginManagerDisabled
	}

	// ищем все .lua файлы в директории
	pattern := filepath.Join(dir, "*.lua")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob plugin directory: %w", err)
	}

	var loadErr error

	// загружаем каждый найденный файл
	for _, path := range matches {
		if err := m.LoadPlugin(path); err != nil {
			// если хотя бы один плагин не загрузился, возвращаем ошибку
			// это позволяет отловить проблемы на старте приложения
			loadErr = errors.Join(loadErr, fmt.Errorf("failed to load plugin %s: %w", path, err))
		}
	}

	return loadErr
}

// UnloadPlugin выгружает плагин по имени.
// Вызывает Cleanup для освобождения ресурсов и удаляет плагин из системы.
//
// Параметры:
//   - name: имя плагина для выгрузки
//
// Возвращает ошибку если:
//   - плагин с таким именем не найден
//   - не удалось вызвать Cleanup
//
// После выгрузки плагин больше недоступен через GetPlugin.
func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enable {
		return ErrPluginManagerDisabled
	}

	// ищем плагин в карте
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// вызываем Cleanup для освобождения ресурсов
	// (закрытие соединений, Lua VM и т.д.)
	if err := plugin.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup plugin: %w", err)
	}

	// удаляем плагин из карты
	delete(m.plugins, name)

	return nil
}

// ReloadPlugin перезагружает плагин без остановки системы.
// Полезно для применения изменений в Lua-скрипте без перезапуска приложения.
//
// Параметры:
//   - name: имя плагина для перезагрузки
//
// Процесс перезагрузки:
//  1. Сохраняем путь к файлу плагина
//  2. Вызываем Cleanup для старого экземпляра
//  3. Создаем новый экземпляр из файла
//  4. Регистрируем новый экземпляр под тем же именем
//
// Возвращает ошибку если плагин не найден или не удалось загрузить новую версию.
func (m *Manager) ReloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enable {
		return ErrPluginManagerDisabled
	}

	// ищем плагин для перезагрузки
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// получаем путь к файлу до очистки
	// нужен для загрузки новой версии
	luaPlugin, ok := plugin.(*LuaPlugin)
	if !ok {
		return fmt.Errorf("plugin %s is not a Lua plugin", name)
	}

	filePath := luaPlugin.filePath

	// очищаем старый плагин (закрываем VM и ресурсы)
	if err := plugin.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup plugin: %w", err)
	}

	// создаем конфигурацию runtime из AllowedModules
	runtimeCfg := &RuntimeConfig{
		AllowedModules: m.config.AllowedModules,
		CallStackSize:  256,
		RegistrySize:   256,
	}

	// загружаем новую версию плагина из файла
	newPlugin, err := NewLuaPluginWithConfig(filePath, runtimeCfg, m.config.Stdlib)
	if err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	// заменяем старый плагин новым
	m.plugins[name] = newPlugin

	return nil
}

// GetPlugin возвращает плагин по имени для использования.
// Используется для получения доступа к плагину из других частей системы.
//
// Параметры:
//   - name: имя плагина
//
// Возвращает:
//   - Plugin интерфейс для вызова Execute, Validate и т.д.
//   - ошибку если плагин не найден
//
// Потокобезопасна - можно вызывать из разных горутин одновременно.
func (m *Manager) GetPlugin(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.config.Enable {
		return nil, ErrPluginManagerDisabled
	}

	// ищем плагин в карте
	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// ListPlugins возвращает информацию о всех загруженных плагинах.
// Используется для отображения списка плагинов в UI, мониторинге и отладке.
//
// Возвращает слайс PluginInfo со всей метаинформацией:
//   - имя, версия, описание, автор
//   - путь к файлу
//   - статус работоспособности
//   - статистику выполнения
//
// Потокобезопасна - можно вызывать параллельно с другими операциями.
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// создаем слайс с правильной емкостью для оптимизации
	infos := make([]PluginInfo, 0, len(m.plugins))

	// собираем информацию о каждом плагине
	for _, plugin := range m.plugins {
		// приводим к *LuaPlugin для доступа к дополнительным полям
		luaPlugin, ok := plugin.(*LuaPlugin)
		if !ok {
			continue // пропускаем если это не Lua плагин
		}

		// формируем структуру с полной информацией
		infos = append(infos, PluginInfo{
			Name:        plugin.Name(),
			Version:     plugin.Version(),
			Description: plugin.Description(),
			Author:      luaPlugin.author,
			FilePath:    luaPlugin.filePath,
			Healthy:     plugin.IsHealthy(),
		})
	}

	return infos
}

// Shutdown корректно останавливает все плагины перед завершением работы.
// Должна вызываться при остановке приложения для graceful shutdown.
//
// Процесс:
//  1. Вызывает Cleanup() для каждого плагина
//  2. Собирает ошибки если они есть
//  3. Очищает карту плагинов
//
// Возвращает последнюю ошибку если хотя бы один Cleanup завершился неудачно.
// Даже при ошибках все плагины будут обработаны и выгружены.
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enable {
		return ErrPluginManagerDisabled
	}

	var lastErr error
	// пытаемся корректно остановить каждый плагин
	for name, plugin := range m.plugins {
		if err := plugin.Cleanup(); err != nil {
			// сохраняем ошибку но продолжаем останавливать остальные
			lastErr = fmt.Errorf("failed to cleanup plugin %s: %w", name, err)
		}
	}

	// очищаем карту плагинов
	m.plugins = make(map[string]Plugin)

	return lastErr
}

// autoLoad автоматически загружает плагины из config.Plugins.PluginsDir.
// срабатывает только при условии config.Plugins.Enable = true.
func (m *Manager) autoLoad() error {
	if !m.config.Enable {
		return nil
	}
	return m.LoadPluginsFromDir(m.config.PluginsDir)
}

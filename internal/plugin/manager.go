package plugins

import (
	"context"
	"fmt"
	"support_bot/internal/plugin/stdlib"
	"sync"
)

type PluginLoader interface {
	LoadByName(ctx context.Context, name string) (LuaPluginDTO, error)
	LoadByNameAndVersion(ctx context.Context, name string, version string) (LuaPluginDTO, error)
	LoadByID(ctx context.Context, id int) (LuaPluginDTO, error)
	LoadByNameAll(ctx context.Context, name string) ([]LuaPluginDTO, error)
}

// Manager управляет жизненным циклом всех загруженных плагинов.
// Обеспечивает потокобезопасную загрузку, выгрузку и доступ к плагинам. //
// Основные функции:
//   - Регистрация и хранение плагинов по имени
//   - Предоставление доступа к плагинам
//   - Перезагрузка плагинов без остановки системы
//   - Graceful shutdown всех плагинов
type Manager struct {
	config *Config      // конфигурация системы плагинов
	mu     sync.RWMutex // мьютекс для потокобезопасного доступа

	repo PluginLoader

	plugSTD *stdlib.STD
}

// NewManager создает новый менеджер плагинов с заданной конфигурацией. //
// Параметры:
//   - cfg: конфигурация плагинов (enable, plugins_dir, таймауты и т.д.)
//
// Возвращает готовый к использованию Manager с пустой картой плагинов.
func NewManager(cfg *Config, plugRepo PluginLoader, std *stdlib.STD) *Manager {
	return &Manager{
		config:  cfg,
		plugSTD: std,
		repo:    plugRepo,
	}
}

func (m *Manager) NewPlugin(ctx context.Context, name string, version *string) (*LuaPlugin, error) {
	var plug LuaPluginDTO
	var err error
	if version != nil {
		plug, err = m.repo.LoadByNameAndVersion(ctx, name, *version)
		if err != nil {
			return new(LuaPlugin), fmt.Errorf("error loading plugin by name and ver: %w", err)
		}

		return m.BuildPluginFromDTO(plug)
	}
	plug, err = m.repo.LoadByName(ctx, name)
	if err != nil {
		return new(LuaPlugin), fmt.Errorf("error loading plugin by name: %w", err)
	}

	return m.BuildPluginFromDTO(plug)
}

func (m *Manager) BuildPluginFromDTO(plug LuaPluginDTO) (*LuaPlugin, error) {
	// создаем конфигурацию runtime из AllowedModules
	runtimeCfg := &RuntimeConfig{
		AllowedModules: m.config.AllowedModules,
		CallStackSize:  256,
		RegistrySize:   256,
	}

	// создаем новый экземпляр Lua-плагина
	// при этом загружается и выполняется Lua-скрипт
	plugin, err := NewLuaPluginWithConfigFromString(plug.PluginStr, runtimeCfg, m.plugSTD)
	if err != nil {
		return new(LuaPlugin), fmt.Errorf("failed to create plugin: %w", err)
	}

	return plugin, nil
}

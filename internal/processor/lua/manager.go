package lua

import (
	"context"
	"fmt"
	"sync"

	"support_bot/internal/processor/lua/stdlib"
)

type PluginProvider interface {
	GetByName(ctx context.Context, name string) (string, error)
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

	provider PluginProvider

	plugSTD *stdlib.STD
}

// NewManager создает новый менеджер плагинов с заданной конфигурацией. //
// Параметры:
//   - cfg: конфигурация плагинов (enable, plugins_dir, таймауты и т.д.)
//
// Возвращает готовый к использованию Manager с пустой картой плагинов.
func NewManager(cfg *Config, provider PluginProvider, std *stdlib.STD) *Manager {
	return &Manager{
		config:   cfg,
		provider: provider,
		plugSTD:  std,
	}
}

func (m *Manager) NewPlugin(plug string) (*LuaPlugin, error) {
	// создаем конфигурацию runtime из AllowedModules
	runtimeCfg := &RuntimeConfig{
		AllowedModules: m.config.AllowedModules,
		CallStackSize:  256,
		RegistrySize:   256,
	}

	// создаем новый экземпляр Lua-плагина
	// при этом загружается и выполняется Lua-скрипт
	plugin, err := NewLuaPlugin(plug, runtimeCfg, m.plugSTD)
	if err != nil {
		return new(LuaPlugin), fmt.Errorf("failed to create plugin: %w", err)
	}

	return plugin, nil
}

func (m *Manager) NewPluginFromName(ctx context.Context, name string) (*LuaPlugin, error) {
	plugStr, err := m.provider.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return m.NewPlugin(plugStr)
}

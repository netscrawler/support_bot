package plugins

import (
	"time"

	"support_bot/internal/plugin/stdlib"
)

// Config конфигурация системы Lua-плагинов
// все интеграции с внешними источниками данных (Metabase, API и т.д.)
// реализуются через Lua-плагины для гибкости и расширяемости.
type Config struct {
	Enable           bool          `yaml:"enable"            env:"PLUGINS_ENABLE"`            // включить систему плагинов
	PluginsDir       string        `yaml:"plugins_dir"       env:"PLUGINS_DIR"`               // директория с Lua-скриптами
	LoadTimeout      time.Duration `yaml:"load_timeout"      env:"PLUGINS_LOAD_TIMEOUT"`      // таймаут загрузки плагина
	ExecutionTimeout time.Duration `yaml:"execution_timeout" env:"PLUGINS_EXECUTION_TIMEOUT"` // таймаут выполнения плагина
	MaxMemoryMB      int           `yaml:"max_memory_mb"     env:"PLUGINS_MAX_MEMORY_MB"`     // лимит памяти для одного плагина
	AllowedModules   []string      `yaml:"allowed_modules"   env:"PLUGINS_ALLOWED_MODULES"`   // разрешенные Lua-модули (http, json и т.д.)
	
	// Stdlib стандартная библиотека для плагинов (опционально)
	Stdlib *stdlib.STD
}

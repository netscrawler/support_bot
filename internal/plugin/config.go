package plugins

import (
	"time"

	"support_bot/internal/plugin/stdlib"
)

// Config конфигурация системы Lua-плагинов
// все интеграции с внешними источниками данных (Metabase, API и т.д.)
// реализуются через Lua-плагины для гибкости и расширяемости.
type Config struct {
	Enable           bool          `yaml:"enable"            env:"PLUGINS_ENABLE"            comment:"Определеяет будет ли система плагинов использоваться в системе"` // включить систему плагинов
	PluginsDir       string        `yaml:"plugins_dir"       env:"PLUGINS_DIR"               comment:"Путь к директории с плагинами"`                                  // директория с Lua-скриптами
	LoadTimeout      time.Duration `yaml:"load_timeout"      env:"PLUGINS_LOAD_TIMEOUT"      comment:"Таймаут для загрузки плагина"`                                   // таймаут загрузки плагина
	ExecutionTimeout time.Duration `yaml:"execution_timeout" env:"PLUGINS_EXECUTION_TIMEOUT" comment:"Таймаут для выполнения плагина."`                                // таймаут выполнения плагина
	MaxMemoryMB      int           `yaml:"max_memory_mb"     env:"PLUGINS_MAX_MEMORY_MB"     comment:"Лимит памяти для 1 плагина в MB"`                                // лимит памяти для одного плагина
	AllowedModules   []string      `yaml:"allowed_modules"   env:"PLUGINS_ALLOWED_MODULES"   comment:"Разрешенные луа модули"`                                         // разрешенные Lua-модули (http, json и т.д.)

	// Stdlib стандартная библиотека для плагинов (опционально)
	Stdlib *stdlib.STD
}

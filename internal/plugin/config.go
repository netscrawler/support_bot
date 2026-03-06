package plugins

import (
	"time"
)

// Config конфигурация системы Lua-плагинов
// все интеграции с внешними источниками данных (Metabase, API и т.д.)
// реализуются через Lua-плагины для гибкости и расширяемости.
type Config struct {
	ExecutionTimeout time.Duration `yaml:"execution_timeout" env:"PLUGINS_EXECUTION_TIMEOUT" comment:"Таймаут для выполнения плагина."` // таймаут выполнения плагина
	MaxMemoryMB      int           `yaml:"max_memory_mb"     env:"PLUGINS_MAX_MEMORY_MB"     comment:"Лимит памяти для 1 плагина в MB"` // лимит памяти для одного плагина
	AllowedModules   []string      `yaml:"allowed_modules"   env:"PLUGINS_ALLOWED_MODULES"   comment:"Разрешенные луа модули"`          // разрешенные Lua-модули (http, json и т.д.)
}

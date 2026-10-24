package lua

import (
	"time"
)

// Config конфигурация системы Lua-плагинов
// все интеграции с внешними источниками данных (Metabase, API и т.д.)
// реализуются через Lua-плагины для гибкости и расширяемости.
type Config struct {
	ExecutionTimeout time.Duration `comment:"Таймаут для выполнения плагина."                            env:"PLUGINS_EXECUTION_TIMEOUT" yaml:"execution_timeout"` // таймаут выполнения плагина
	MaxMemoryMB      int           `comment:"Лимит памяти для 1 плагина в MB"                            env:"PLUGINS_MAX_MEMORY_MB"     yaml:"max_memory_mb"`     // лимит памяти для одного плагина
	AllowedModules   []string      `comment:"Разрешенные луа модули"                                     env:"PLUGINS_ALLOWED_MODULES"   yaml:"allowed_modules"`   // разрешенные Lua-модули (http, json и т.д.)
}

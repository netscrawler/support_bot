package lua

import "errors"

var (
	// ErrPluginTableNotFound возвращается когда глобальная таблица plugin не найдена в Lua скрипте.
	ErrPluginTableNotFound = errors.New("plugin table not found")

	// ErrPluginNameRequired возвращается когда имя плагина пустое или не указано.
	ErrPluginNameRequired = errors.New("plugin name is required")

	// ErrPluginInitFailed возвращается когда инициализация плагина завершилась неудачей.
	ErrPluginInitFailed = errors.New("plugin init failed")

	// ErrValidationFailed возвращается когда валидация параметров не прошла.
	ErrValidationFailed = errors.New("validation failed")
)

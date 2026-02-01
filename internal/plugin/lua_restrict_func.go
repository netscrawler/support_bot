package plugins

var RestrictedModules = map[string]bool{
	// опасные глобальные функции
	"dofile":     true,
	"loadfile":   true,
	"load":       true,
	"loadstring": true,
	"module":     true,
	"setfenv":    true,
	"getfenv":    true,

	// опасные стандартные модули
	"debug":   true, // полный доступ к VM
	"io":      true, // работа с файлами
	"os":      true, // нужно фильтровать функции внутри
	"package": true, // управление загрузкой модулей
}

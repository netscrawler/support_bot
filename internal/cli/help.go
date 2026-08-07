package cli

import (
	"fmt"
	"os"
)

func Help() {
	fmt.Fprintf(
		os.Stderr,
		"Support TgBot CLI — Инструмент для управления и запуска бота поддержки\n\n",
	)
	fmt.Fprintf(os.Stderr, "Использование:\n")
	fmt.Fprintf(os.Stderr, "  %s <команда> [аргументы]\n\n", os.Args[0])

	fmt.Fprintf(os.Stderr, "Команды:\n")

	fmt.Fprintf(os.Stderr, "  run        Запуск основного процесса бота\n")
	fmt.Fprintf(os.Stderr, "             Аргументы:\n")
	fmt.Fprintf(
		os.Stderr,
		"               -config <путь>  Путь к файлу конфигурации (YAML или .env)\n\n",
	)

	fmt.Fprintf(os.Stderr, "  config     Управление конфигурацией приложения\n")
	fmt.Fprintf(os.Stderr, "             Подкоманды:\n")
	fmt.Fprintf(os.Stderr, "               validate  Проверка файла конфигурации на ошибки\n")
	fmt.Fprintf(os.Stderr, "                         -config <путь>  Путь к проверяемому файлу\n")
	fmt.Fprintf(
		os.Stderr,
		"               update    Перезапись конфигурации (нормализация структуры)\n",
	)
	fmt.Fprintf(os.Stderr, "                         -config <путь>  Путь к обновляемому файлу\n")
	fmt.Fprintf(os.Stderr, "               create    Генерация файла конфигурации по умолчанию\n")
	fmt.Fprintf(
		os.Stderr,
		"                         -out <путь>     Куда сохранить (по умолчанию в консоль)\n",
	)
	fmt.Fprintf(
		os.Stderr,
		"                         -format <тип>   Формат: yaml или env (по умолчанию yaml)\n\n",
	)

	fmt.Fprintf(os.Stderr, "  ctl        Управление данными (отчетами и скриптами) в БД\n")
	fmt.Fprintf(os.Stderr, "             Подкоманды:\n")
	fmt.Fprintf(os.Stderr, "               export    Экспорт всех отчетов из БД в JSON-файлы\n")
	fmt.Fprintf(os.Stderr, "                         -config <путь>  Конфиг для подключения к БД\n")
	fmt.Fprintf(
		os.Stderr,
		"                         -out <папка>    Директория для сохранения файлов\n",
	)
	fmt.Fprintf(os.Stderr, "               create    Создание JSON-шаблона для нового отчета\n")
	fmt.Fprintf(
		os.Stderr,
		"                         -out <путь>     Файл для записи (по умолчанию stdout)\n",
	)
	fmt.Fprintf(os.Stderr, "               apply     Загрузка отчетов из файлов в базу данных\n")
	fmt.Fprintf(os.Stderr, "                         -config <путь>  Конфиг для подключения к БД\n")
	fmt.Fprintf(
		os.Stderr,
		"                         -report <путь>  Путь к файлу или маска (например, 'reports/*.json')\n",
	)
	fmt.Fprintf(
		os.Stderr,
		"               validate  Проверка структуры JSON-файлов отчетов без загрузки в БД\n",
	)
	fmt.Fprintf(os.Stderr, "                         -report <путь>  Путь к файлу или маска\n")
	fmt.Fprintf(os.Stderr, "               script    Работа с Lua-скриптами (плагинами)\n")
	fmt.Fprintf(os.Stderr, "                         create -out <путь>  Создать пример скрипта\n")
	fmt.Fprintf(
		os.Stderr,
		"                         save   -config <путь> -script <путь>  Сохранить скрипт в БД\n\n",
	)

	fmt.Fprintf(
		os.Stderr,
		"  version    Вывод версии приложения, хэша коммита и времени сборки\n\n",
	)

	fmt.Fprintf(os.Stderr, "Совет:\n")
	fmt.Fprintf(
		os.Stderr,
		"  Для каждой подкоманды можно вызвать справку, например: %s ctl export --help\n",
		os.Args[0],
	)
}

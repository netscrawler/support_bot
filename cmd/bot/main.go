package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"support_bot/internal/app"
	"support_bot/internal/config"
	"support_bot/internal/pkg"
	"support_bot/internal/pkg/logger"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	Version   = "v0.0.0"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	switch modeStart() {
	case helpMode:
		help()

		return
	case verMode:
		version()

		return
	case createEnvMode:
		createEnv()

		return
	case createYamlMode:
		createConf()

		return
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	validErr := cfg.Validate()
	if validErr != nil {
		panic(validErr)
	}

	log, err := logger.Setup(cfg.Log)
	log.Error("config creating error", slog.Any("error", err))

	ctx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	log.Debug(
		"starting with config",
		slog.Any("config", cfg),
		slog.GroupAttrs("app_info", slog.Any("version", Version),
			slog.Any("commit", Commit),
			slog.Any("BuildTime", BuildTime)),
	)

	app, err := app.New(ctx, cfg)
	if err != nil {
		log.Error("failing creating app", slog.Any("error", err))

		return
	}

	err = app.Start(ctx)
	if err != nil {
		log.Error("failing start app", slog.Any("error", err))

		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	log.Info("receive stop signal", slog.Any("finish time", 10*time.Second))

	sCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownCtx := logger.AppendCtx(sCtx,
		slog.Any("function", "shutting down"))
	app.GracefulShutdown(shutdownCtx)
}

var (
	SetModeVer          = false
	SetModeHelp         = false
	SetModeCreateConfig = false
	SetModeCreateEnv    = false
)

type mode int

const (
	commonMode mode = iota
	helpMode
	verMode
	createEnvMode
	createYamlMode
)

func modeStart() mode {
	flag.BoolVar(&SetModeVer, "v", false, "Версия приложения")
	flag.BoolVar(&SetModeHelp, "h", false, "Помощь")
	flag.StringVar(&config.ConfigPath, "config", "", "Путь к файлу конфигурации")
	flag.BoolVar(
		&SetModeCreateConfig,
		"example-config",
		false,
		"Сгенерировать пример файла конфигурации",
	)
	flag.BoolVar(
		&SetModeCreateEnv,
		"example-env",
		false,
		"Сгенерировать пример файла .env",
	)
	flag.Parse()

	if SetModeVer {
		return verMode
	}

	if SetModeHelp {
		return helpMode
	}

	if SetModeCreateEnv {
		return createEnvMode
	}

	if SetModeCreateConfig {
		return createYamlMode
	}

	return commonMode
}

func version() {
	fmt.Printf(
		"Version: %s\nCommit: %s\nBuildTime: %s\nRuntime: %s",
		Version,
		Commit,
		BuildTime,
		runtime.Version(),
	)
}

func createEnv() {
	defaultEnv := config.Default()

	defEnv, err := pkg.GenerateEnv(defaultEnv, "", "")
	if err != nil {
		fmt.Printf("Unable to create env: %s", err.Error())

		return
	}

	fmt.Println("# Пример .env файла")
	fmt.Println("# Создайте файл .env и поместите туда, заменив значения на свои")
	fmt.Println(defEnv)
}

func createConf() {
	defaultConf := config.Default()

	node, err := pkg.StructToYAMLNode(defaultConf)
	if err != nil {
		fmt.Printf("Unable to create config: %s", err.Error())

		return
	}

	mCfg, err := yaml.Marshal(node)
	if err != nil {
		fmt.Printf("Unable to create config: %s", err.Error())

		return
	}

	fmt.Println("# example config")
	fmt.Println("# Создайте файл config.yaml и поместите туда, заменив значения на свои")
	fmt.Println(string(mCfg))
}

func help() {
	fmt.Println(`
Support Bot CLI
===============

Использование:
  support_bot [опции]

Основные флаги:
  -h
        Помощь
  -v
        Показать версию приложения
  -config string
        Путь к файлу конфигурации
  -example-config
        Сгенерировать пример YAML конфигурации
  -example-env
        Сгенерировать пример .env файла

Примеры запуска:
  # Запуск с указанным конфигом
  support_bot --config=config.yaml

  # Вывод версии
  support_bot -v

  # Генерация example YAML
  support_bot -example-config > config.yaml

  # Генерация example .env
  support_bot -example-env > .env`)

	// Также можно напечатать все флаги автоматически:
	fmt.Println("Доступные флаги и их описания:")
	flag.PrintDefaults()
}

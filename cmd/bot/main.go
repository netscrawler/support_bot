package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"support_bot/internal/app"
	"support_bot/internal/config"
	"support_bot/internal/pkg"
	"support_bot/internal/pkg/logger"

	"gopkg.in/yaml.v3"
)

var (
	Version   = "v0.0.0"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	modeStart()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	validErr := cfg.Validate()
	if validErr != nil {
		panic(validErr)
	}

	log, err := logger.Setup(cfg.Log)
	if err != nil {
		log.Error("log creating error", slog.Any("error", err))
	}

	ctx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	log.Debug(
		"starting with config",
		slog.Any("config", cfg),
		slog.GroupAttrs("app_info", slog.Any("version", Version),
			slog.Any("commit", commit),
			slog.Any("BuildTime", buildTime)),
	)

	appContainer, err := app.New(ctx, cfg)
	if err != nil {
		log.Error("failing creating appContainer", slog.Any("error", err))

		return
	}

	err = appContainer.Start(ctx)
	if err != nil {
		log.Error("failing start appContainer", slog.Any("error", err))

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
	appContainer.GracefulShutdown(shutdownCtx)
}

var (
	setModeVer          = false
	setModeHelp         = false
	setModeCreateConfig = false
	setModeCreateEnv    = false
)

func modeStart() {
	flag.BoolVar(&setModeVer, "v", false, "Версия приложения")
	flag.BoolVar(&setModeHelp, "h", false, "Помощь")
	flag.StringVar(&config.Path, "config", "", "Путь к файлу конфигурации")
	flag.BoolVar(
		&setModeCreateConfig,
		"example-config",
		false,
		"Сгенерировать пример файла конфигурации",
	)
	flag.BoolVar(
		&setModeCreateEnv,
		"example-env",
		false,
		"Сгенерировать пример файла .env",
	)
	flag.Parse()

	if setModeVer {
		version()
	}

	if setModeHelp {
		help()
	}

	if setModeCreateEnv {
		createEnv()
	}

	if setModeCreateConfig {
		createConf()
	}
}

func version() {
	fmt.Printf(
		"Version: %s\nCommit: %s\nBuildTime: %s\nRuntime: %s",
		Version,
		commit,
		buildTime,
		runtime.Version(),
	)
	os.Exit(0)
}

func createEnv() {
	defaultEnv := config.Default()

	defEnv, err := pkg.GenerateEnv(defaultEnv, "", "")
	if err != nil {
		fmt.Printf("Unable to create env: %s", err.Error())

		os.Exit(1)
	}

	fmt.Println("# Пример .env файла")
	fmt.Println("# Создайте файл .env и поместите туда, заменив значения на свои")
	fmt.Println(defEnv)
	os.Exit(0)
}

func createConf() {
	defaultConf := config.Default()

	node, err := pkg.StructToYAMLNode(defaultConf)
	if err != nil {
		fmt.Printf("Unable to create config: %s", err.Error())
		os.Exit(1)
	}

	mCfg, err := yaml.Marshal(node)
	if err != nil {
		fmt.Printf("Unable to create config: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("# example config")
	fmt.Println("# Создайте файл config.yaml и поместите туда, заменив значения на свои")
	fmt.Println(string(mCfg))
	os.Exit(0)
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
	os.Exit(0)
}

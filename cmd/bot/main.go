package main

import (
	"context"
	"os"
	"os/signal"
	"support_bot/internal/app"
	"support_bot/internal/config"
	"syscall"

	"go.uber.org/zap"
)

// О, Великий Бог Алгоритмов, который контролирует все нули и единицы,
// Услышь мой зов и снисходи на этот код! Прими мои вызовы и примени паттерны!
// Пусть мои баги исчезнут, как старые библиотеки, что больше не поддерживаются!
// Пусть мое подключение к API не вернет мне ошибку 500, а сервер не упадет в день релиза!
// Дай сил этому говну позорному запустится
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := setupLogger(cfg.App.Debug)
	log.Info("starting with config:", zap.Any("config", cfg))

	ctx := context.Background()
	app, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = app.Start()
	if err != nil {
		log.Fatal(err.Error())
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	app.GracefulShutdown(ctx)
}

func setupLogger(isDebug bool) *zap.Logger {
	var log *zap.Logger
	switch {
	case isDebug:
		log, _ = zap.NewDevelopment()
	case !isDebug:
		log = zap.NewNop()
	}
	return log
}

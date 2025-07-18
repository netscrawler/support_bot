package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"support_bot/internal/app"
	"support_bot/internal/config"

	"go.uber.org/zap"
)

const (
	debug string = "debug"
	prod  string = "prod"
)

// Дай сил этому говну позорному запустится.
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := setupLogger(cfg.LogLevel)

	ctx := context.Background()

	app, err := app.New(ctx, cfg)
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

func setupLogger(isDebug string) *zap.Logger {
	var log *zap.Logger

	switch isDebug {
	case debug:
		log, _ = zap.NewDevelopment()
	case prod:
		log, _ = zap.NewProduction()
	default:
		log, _ = zap.NewDevelopment()
	}

	return log
}

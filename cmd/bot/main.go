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

// Дай сил этому говну позорному запустится.
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := setupLogger(cfg.App.Debug)

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

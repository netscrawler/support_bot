package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"support_bot/internal/app"
	"support_bot/internal/config"
	"support_bot/internal/pkg/logger"
	"syscall"
	"time"
)

var (
	Version   = "v0.0.0"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Дай сил этому говну позорному запустится.
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.Setup(cfg.Log)

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

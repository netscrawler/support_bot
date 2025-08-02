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

	ctx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	log.Debug("starting with config", slog.Any("config", cfg))

	app, err := app.New(ctx, cfg)
	if err != nil {
		log.Error("failing creating app", slog.Any("error", err))
		os.Exit(1)
	}

	err = app.Start(ctx)
	if err != nil {
		log.Error("failing start app", slog.Any("error", err))
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	sCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownCtx := logger.AppendCtx(sCtx,
		slog.Any("function", "shutting down"))
	app.GracefulShutdown(shutdownCtx)
}

func setupLogger(logLevel string) *slog.Logger {
	var log *slog.Logger
	var opts *slog.HandlerOptions

	switch logLevel {
	case debug:
		opts = &slog.HandlerOptions{Level: slog.LevelDebug}
	case prod:
		opts = &slog.HandlerOptions{Level: slog.LevelInfo}
	default:
		opts = &slog.HandlerOptions{Level: slog.LevelInfo}
	}

	log = slog.New(
		logger.ContextHandler{
			Handler: slog.NewTextHandler(
				os.Stdout,
				opts,
			),
		},
	)

	slog.SetDefault(log)

	return log
}

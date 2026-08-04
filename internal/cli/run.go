package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"support_bot/internal/app"
	"support_bot/internal/config"
	"support_bot/internal/pkg/logger"
)

func Run(version, commit, buildTime string, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "Путь до конфига")

	err := fs.Parse(args)
	if err != nil {
		return err
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	validErr := cfg.Validate()
	if validErr != nil {
		return fmt.Errorf("validate config: %w", validErr)
	}

	log, err := logger.Setup(cfg.Log)
	if err != nil {
		return fmt.Errorf("setup logger: %w", err)
	}

	ctx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	log.Info(
		"starting with config",
		slog.Any("config", cfg),
		slog.GroupAttrs("app_info", slog.Any("version", version),
			slog.Any("commit", commit),
			slog.Any("BuildTime", buildTime)),
	)

	appContainer, err := app.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	err = appContainer.Start(ctx)
	if err != nil {
		return fmt.Errorf("start app: %w", err)
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
	return nil
}

package cli

import (
	"flag"
	"fmt"
	"log/slog"
	"support_bot/internal/app"
	"support_bot/internal/config"
	"support_bot/internal/pkg/logger"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
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

	log.Info(
		"starting with config",
		slog.Any("config", cfg),
		slog.GroupAttrs("app_info", slog.Any("version", version),
			slog.Any("commit", commit),
			slog.Any("BuildTime", buildTime)),
	)

	fxApp := fx.New(
		app.Module,
		fx.Supply(cfg, log),
		fx.StopTimeout(cfg.Timeout.Shutdown),
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
	)
	if err := fxApp.Err(); err != nil {
		return fmt.Errorf("create app: %w", err)
	}
	fxApp.Run()
	return nil
}

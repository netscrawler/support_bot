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
		// Known gap: if construction failed partway through (some providers'
		// OnStop hooks already registered, e.g. a DB connection), those
		// resources are not cleaned up here. Calling fxApp.Stop() would not
		// help either — fx's internal lifecycle only runs OnStop hooks whose
		// OnStart phase (or the reverse-order walk gated by having reached
		// the "started"/"starting" state) actually ran, and that only
		// happens via App.Start()/App.Run(), which we never reach on this
		// path (verified against go.uber.org/fx v1.24.0's
		// internal/lifecycle.Lifecycle.Stop: it no-ops unless the lifecycle
		// state is started/incompleteStart/starting, and numStarted is 0
		// here since Start() was never called). Low practical impact: the
		// process exits immediately after this return.
		return fmt.Errorf("create app: %w", err)
	}
	// fx.App.Run() installs OS signal handlers and blocks until shutdown.
	// On a start-hook failure it logs via the fxevent logger and calls
	// os.Exit(1) directly, bypassing this function's normal error-return
	// path — this is fx's standard Run() behavior, not a bug.
	fxApp.Run()
	return nil
}

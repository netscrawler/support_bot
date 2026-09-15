package app

import (
	"context"
	"log/slog"
	"support_bot/internal/config"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"

	"go.uber.org/fx"
)

var storageModule = fx.Module("storage", fx.Provide(newAppContext, newDB))

func newAppContext(lc fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}

func newDB(ctx context.Context, cfg *config.Config, log *slog.Logger, lc fx.Lifecycle) (*postgres.DB, error) {
	connCtx, cancel := context.WithTimeout(ctx, cfg.Database.DatabaseConnect)
	defer cancel()

	connCtx = logger.AppendCtx(connCtx, slog.Any("function", "connecting to database"))

	rdb, err := postgres.New(connCtx, cfg.Database, log)
	if err != nil {
		log.ErrorContext(connCtx, "unable to create connection", slog.Any("error", err))

		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return rdb.Stop(ctx)
		},
	})

	return rdb, nil
}

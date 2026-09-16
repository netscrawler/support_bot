// Package postgres
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool   *pgxpool.Pool
	cancel context.CancelFunc

	log *slog.Logger
}

func parsePoolConfig(cfg Config) (*pgxpool.Config, error) {
	pgxCfg, err := pgxpool.ParseConfig(cfg.getDSN())
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pgxCfg.MaxConns = int32(cfg.MaxConns) //nolint:gosec // trusted config value
	pgxCfg.MaxConnLifetime = cfg.MaxConnLifeTime
	pgxCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	// ponytail: pgxpool has no direct "max idle conns count" knob the way
	// database/sql's SetMaxIdleConns does — only MaxConns + MaxConnIdleTime.
	// cfg.MaxIdleConns is intentionally not mapped here; revisit if pool
	// behavior under load needs a closer idle-count analog than pgxpool
	// exposes today.

	return pgxCfg, nil
}

func New(ctx context.Context, cfg Config, log *slog.Logger) (*DB, error) {
	l := log.With(slog.Any("module", "postgres"))

	l.InfoContext(ctx, "start connecting to postgres")

	pgxCfg, err := parsePoolConfig(cfg)
	if err != nil {
		l.ErrorContext(ctx, "error building pool config", slog.Any("error", err))

		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		l.ErrorContext(ctx, "error connecting to database", slog.Any("error", err))

		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	l.InfoContext(
		ctx,
		"database connection established",
		slog.Group(
			"config",
			slog.Any("max conns", cfg.MaxConns),
			slog.Any("max conn lifetime", cfg.MaxConnLifeTime),
			slog.Any("max conn idle time", cfg.MaxConnIdleTime),
		),
	)

	cctx, cancel := context.WithCancel(context.Background())

	d := &DB{
		pool:   pool,
		cancel: cancel,
		log:    l,
	}

	d.startMonitor(cctx)

	return d, nil
}

func (d *DB) GetConn() *pgxpool.Pool {
	return d.pool
}

func (d *DB) Stop(_ context.Context) error {
	d.cancel()
	d.pool.Close()

	return nil
}

func (d *DB) startMonitor(ctx context.Context) {
	go func() {
		log := d.log.With(slog.Any("submodule", "monitor"))

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.InfoContext(ctx, "DB monitor stopped", slog.String("reason", ctx.Err().Error()))

				return

			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := d.pool.Ping(pingCtx)

				cancel()

				if err != nil {
					log.WarnContext(ctx, "DB ping failed", slog.Any("error", err))
				}
			}
		}
	}()
}

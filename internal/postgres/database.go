// Package postgres
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	// sqlx driver.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	db     *sqlx.DB
	cancel context.CancelFunc

	log *slog.Logger
}

func New(ctx context.Context, cfg PostgresConfig, log *slog.Logger) (*DB, error) {
	l := log.With(slog.Any("module", "postgres"))

	l.InfoContext(ctx, "start connecting to postgres")

	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.GetDSN())
	if err != nil {
		l.ErrorContext(ctx, "error connecting to database", slog.Any("error", err))

		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MaxConnLifeTime)
	db.SetConnMaxIdleTime(cfg.MaxConnIdleTime)

	if err := db.Ping(); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			l.ErrorContext(ctx, "unable close connection correctly", slog.Any("error", err))
		}

		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	l.InfoContext(
		ctx,
		"database connection established",
		slog.Group(
			"config",
			slog.Any("max open conns", cfg.MaxConns),
			slog.Any("max idle conns", cfg.MaxIdleConns),
			slog.Any("max conn lifetime", cfg.MaxConnLifeTime),
			slog.Any("max conn idle time", cfg.MaxConnIdleTime),
		),
	)

	cctx, cancel := context.WithCancel(context.Background())

	d := &DB{
		db:     db,
		cancel: cancel,
		log:    l,
	}

	d.startMonitor(cctx)

	return d, nil
}

func (d *DB) GetConn() *sqlx.DB {
	return d.db
}

func (d *DB) Stop(_ context.Context) error {
	d.cancel()

	return d.db.Close()
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
				err := d.db.PingContext(pingCtx)

				cancel()

				if err != nil {
					log.WarnContext(ctx, "DB ping failed", slog.Any("error", err))
				}
			}
		}
	}()
}

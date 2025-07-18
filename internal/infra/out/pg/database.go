package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

func New(ctx context.Context, connStr string) (*pgx.Conn, error) {
	const op = "storage.postgres.Init"

	log := slog.Default()

	var err error

	connCfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		log.Error("%s unable to ParseConfig")

		return nil, err
	}

	conn, err := pgx.ConnectConfig(ctx, connCfg)
	if err != nil {
		log.Error(op, slog.Any("error", err))

		return nil, fmt.Errorf("%s : %w", op, err)
	}

	if err := conn.Ping(ctx); err != nil {
		log.Error(op, slog.Any("error", err))

		return nil, fmt.Errorf("%s : %w", op, err)
	}

	log.Info(op + " : successfully connected")

	return conn, nil
}

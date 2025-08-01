// Package postgres
package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

func New(ctx context.Context, connStr string) (*pgx.Conn, error) {
	l := slog.Default()

	l.DebugContext(ctx, "connecting to: ", slog.String("DSN", connStr))

	connCfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	conn, err := pgx.ConnectConfig(ctx, connCfg)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database unreacheble: %w", err)
	}

	l.InfoContext(ctx, "successfully connected")

	return conn, nil
}

// Package postgres
package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

type ReconnectableDB struct {
	getConn func() *pgx.Conn
}

func NewReconnectableDB(fn func() *pgx.Conn) *ReconnectableDB {
	return &ReconnectableDB{
		getConn: fn,
	}
}

func (r *ReconnectableDB) Exec(
	ctx context.Context,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	return r.getConn().Exec(ctx, sql, args...)
}

func (r *ReconnectableDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return r.getConn().Query(ctx, sql, args...)
}

func (r *ReconnectableDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return r.getConn().QueryRow(ctx, sql, args...)
}

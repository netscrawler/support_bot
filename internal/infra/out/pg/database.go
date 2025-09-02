// Package postgres
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ReconnectableDB struct {
	conn    *pgx.Conn
	getConn func() *pgx.Conn
	config  *pgx.ConnConfig
	cancel  context.CancelFunc
}

func (rdb *ReconnectableDB) Exec(
	ctx context.Context,
	sql string,
	args ...any,
) (pgconn.CommandTag, error) {
	return rdb.getConn().Exec(ctx, sql, args...)
}

func (rdb *ReconnectableDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return rdb.getConn().Query(ctx, sql, args...)
}

func (rdb *ReconnectableDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return rdb.getConn().QueryRow(ctx, sql, args...)
}

func New(ctx context.Context, connStr string) (*ReconnectableDB, error) {
	l := slog.Default()

	l.DebugContext(ctx, "connecting to: ", slog.String("DSN", connStr))

	connCfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	conn, err := newConn(ctx, connCfg)
	if err != nil {
		return nil, err
	}

	rdb := newReconnectableDB(conn, connCfg)

	return rdb, nil
}

func (rdb *ReconnectableDB) Stop(ctx context.Context) error {
	rdb.cancel()
	return rdb.getConn().Close(ctx)
}

func newConn(ctx context.Context, config *pgx.ConnConfig) (*pgx.Conn, error) {
	l := slog.Default()

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database unreacheble: %w", err)
	}

	l.InfoContext(ctx, "successfully connected")

	return conn, nil
}

func newReconnectableDB(conn *pgx.Conn, connConfig *pgx.ConnConfig) *ReconnectableDB {
	rdb := &ReconnectableDB{conn: conn}
	rdb.config = connConfig
	rdb.getConn = func() *pgx.Conn { return rdb.conn }

	ctx, cancel := context.WithCancel(context.Background())
	rdb.cancel = cancel

	rdb.startDBMonitor(ctx)

	return rdb
}

func (rdb *ReconnectableDB) startDBMonitor(ctx context.Context) {
	go func() {
		log := slog.Default()
		log.InfoContext(ctx, "starting db monitor")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("DB monitor stopped", slog.String("reason", ctx.Err().Error()))
				return
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := rdb.getConn().Ping(pingCtx)
				cancel()

				if err != nil {
					log.Warn("DB ping failed, starting reconnect loop", slog.Any("error", err))
					rdb.reconnectLoop(ctx)
				}
			}
		}
	}()
}

func (rdb *ReconnectableDB) reconnectLoop(ctx context.Context) {
	log := slog.Default()

	const (
		initialDelay = time.Second      // стартовая задержка
		maxDelay     = time.Minute      // максимальная задержка
		timeoutEach  = 10 * time.Second // таймаут на одну попытку пинга или реконнекта
	)

	delay := initialDelay

	for {
		select {
		case <-ctx.Done():
			log.Info("Reconnect loop stopped", slog.String("reason", ctx.Err().Error()))
			return
		default:
			connCtx, cancel := context.WithTimeout(ctx, timeoutEach)
			err := rdb.getConn().Ping(connCtx)
			cancel()

			if err == nil {
				log.Info("Database connection restored")
				return
			}

			log.Warn("Database unavailable", slog.Any("error", err))

			if rdb.getConn().IsClosed() {
				connCtx, cancel := context.WithTimeout(ctx, timeoutEach)
				newConn, err := newConn(connCtx, rdb.config)
				cancel()
				if err == nil {
					rdb.conn = newConn
					log.Info("Successfully reconnected to database")
					return
				}
				log.Warn("Reconnect attempt failed", slog.Any("error", err))
			}

			sleep := time.Duration(rand.Int63n(int64(delay)))
			log.Info("Waiting before next retry", slog.Duration("delay", sleep))
			time.Sleep(sleep)

			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

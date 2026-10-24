// Package postgres выполняет SQL напрямую в Postgres-базах из конфига.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

type target struct {
	pool    txBeginner
	timeout time.Duration
}

type Collector struct {
	dbs   map[string]target
	pools []*pgxpool.Pool
}

func New(ctx context.Context, cfgs map[string]Config, log *slog.Logger) (*Collector, error) {
	c := &Collector{dbs: make(map[string]target, len(cfgs))}

	for name, cfg := range cfgs {
		pc, err := poolConfig(cfg)
		if err != nil {
			c.Close()

			return nil, fmt.Errorf("database %q: %w", name, err)
		}

		pool, err := pgxpool.NewWithConfig(ctx, pc)
		if err != nil {
			c.Close()

			return nil, fmt.Errorf("database %q: %w", name, err)
		}

		c.pools = append(c.pools, pool)
		c.dbs[name] = target{pool: pool, timeout: cfg.withDefaults().StatementTimeout}

		// Бот должен стартовать, даже если база недоступна.
		if err := pool.Ping(ctx); err != nil {
			log.WarnContext(ctx, "postgres collector: ping failed",
				slog.String("database", name), slog.Any("error", err))
		}
	}

	return c, nil
}

func (c *Collector) Close() {
	for _, p := range c.pools {
		p.Close()
	}
}

func (c *Collector) Fetch(
	ctx context.Context,
	db, sql string,
	params map[string]string,
) ([]map[string]any, error) {
	t, ok := c.dbs[db]
	if !ok {
		return nil, fmt.Errorf("unknown database %q", db)
	}

	// Только чтение и никогда Commit: любая запись откатывается (граница безопасности).
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	// SET не принимает bind-параметры, поэтому число подставляем сами.
	if _, err := tx.Exec(
		ctx,
		fmt.Sprintf("SET LOCAL statement_timeout = %d", t.timeout.Milliseconds()),
	); err != nil {
		return nil, fmt.Errorf("set statement_timeout: %w", err)
	}

	args := make(pgx.NamedArgs, len(params))
	for k, v := range params {
		args[k] = v
	}

	rows, err := tx.Query(ctx, sql, args)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	res, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, fmt.Errorf("collect rows: %w", err)
	}

	// Нормализация как у Metabase-коллектора: numeric/time/uuid -> float64/строки.
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal rows: %w", err)
	}

	out := []map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal rows: %w", err)
	}

	return out, nil
}

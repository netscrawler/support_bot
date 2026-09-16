package store

import (
	"context"
	"fmt"
	"support_bot/internal/db/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txBeginner is the minimal seam ExecTx needs — satisfied by *pgxpool.Pool
// in production and by pgxmock's pool in tests.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ExecTx runs fn inside a single transaction, committing on success and
// rolling back on any error. This replaces internal/pkg/uow wholesale.
func ExecTx(ctx context.Context, pool *pgxpool.Pool, fn func(*sqlcgen.Queries) error) error {
	return ExecTxPool(ctx, pool, fn)
}

// ExecTxPool is ExecTx generalized over txBeginner for testability with
// pgxmock; ExecTx is the production-facing name every Store calls.
func ExecTxPool(ctx context.Context, pool txBeginner, fn func(*sqlcgen.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once committed; pgx.ErrTxClosed expected

	if err := fn(sqlcgen.New(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

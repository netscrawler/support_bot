package store

import (
	"context"
	"errors"
	"fmt"

	"support_bot/internal/db/sqlcgen"

	"github.com/jackc/pgx/v5"
)

// txBeginner is the minimal seam ExecTx needs — satisfied by *pgxpool.Pool
// in production and by pgxmock's pool in tests.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ExecTx выполняет fn в рамках одной транзакции, начатой через pool: при
// успехе коммитит, при любой ошибке (включая ошибку из fn) откатывает.
// Полностью заменяет internal/pkg/uow. Параметр pool имеет тип txBeginner,
// а не конкретный *pgxpool.Pool, чтобы тесты могли передавать pgxmock-пул
// напрямую, без реального подключения к БД.
func ExecTx(
	ctx context.Context,
	pool txBeginner,
	fn func(*sqlcgen.Queries) error,
) (resultErr error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		if rollbackErr := tx.Rollback(
			ctx,
		); rollbackErr != nil &&
			!errors.Is(rollbackErr, pgx.ErrTxClosed) {
			resultErr = errors.Join(
				resultErr,
				fmt.Errorf("rollback tx: %w", rollbackErr),
			)
		}
	}()

	if err := fn(sqlcgen.New(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	committed = true

	return nil
}

package store

import (
	"context"
	"errors"
	"support_bot/internal/db/sqlcgen"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestExecTx_BeginFails(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	beginErr := errors.New("connection pool exhausted")
	pool.ExpectBegin().WillReturnError(beginErr)

	called := false
	err = ExecTxPool(context.Background(), pool, func(*sqlcgen.Queries) error {
		called = true
		return nil
	})

	if called {
		t.Fatal("fn was called despite Begin failing")
	}
	if !errors.Is(err, beginErr) {
		t.Fatalf("ExecTx() error = %v, want wrapping %v", err, beginErr)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecTx_FnErrorRollsBack(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectBegin()
	pool.ExpectRollback()

	fnErr := errors.New("boom")
	err = ExecTxPool(context.Background(), pool, func(*sqlcgen.Queries) error {
		return fnErr
	})

	if !errors.Is(err, fnErr) {
		t.Fatalf("ExecTx() error = %v, want %v", err, fnErr)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (commit must not have been called): %v", err)
	}
}

func TestExecTx_CommitsOnSuccess(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectBegin()
	pool.ExpectCommit()
	pool.ExpectRollback() // the deferred rollback after a successful commit is a documented pgx no-op

	err = ExecTxPool(context.Background(), pool, func(*sqlcgen.Queries) error {
		return nil
	})
	if err != nil {
		t.Fatalf("ExecTx() error = %v, want nil", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

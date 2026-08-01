package uow

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type UOW interface {
	Commit() error
	Rollback() error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

type dBUow struct {
	*sqlx.Tx
}

func NewUOW(tx *sqlx.Tx) UOW {
	return &dBUow{tx}
}

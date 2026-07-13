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
}

type dBUow struct {
	*sqlx.Tx
}

func NewUOW(tx *sqlx.Tx) UOW {
	return &dBUow{tx}
}

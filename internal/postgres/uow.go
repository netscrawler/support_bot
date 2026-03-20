package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type txKey string

const txxKey txKey = "tx"

type UOW interface {
	Start(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Uow struct {
	db func() *sqlx.DB
}

func NewUOW(fn func() *sqlx.DB) *Uow {
	return &Uow{
		db: fn,
	}
}

func (u *Uow) Start(ctx context.Context) (context.Context, error) {
	return Start(ctx, u.db())
}

func (u *Uow) Commit(ctx context.Context) error {
	tx, err := u.TxFromCtx(ctx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (u *Uow) Rollback(ctx context.Context) error {
	tx, err := u.TxFromCtx(ctx)
	if err != nil {
		return err
	}
	return tx.Rollback()
}

func Start(ctx context.Context, db *sqlx.DB) (context.Context, error) {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return ctx, err
	}
	txCtx := context.WithValue(ctx, txxKey, tx)

	return txCtx, nil
}

func (u *Uow) TxFromCtx(ctx context.Context) (*sqlx.Tx, error) {
	tx, ok := ctx.Value(txxKey).(*sqlx.Tx)
	if !ok {
		return nil, fmt.Errorf("tx not started")
	}

	return tx, nil
}

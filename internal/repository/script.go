package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Script struct {
	db *sqlx.DB
}

func NewScript(db *sqlx.DB) *Script {
	return &Script{db: db}
}

func (r *Script) Save(ctx context.Context, name string, script string) error {
	const query = "insert into lua_scripts(name, script) values ($1, $2) on conflict (name) do nothing "
	_, err := r.db.ExecContext(ctx, query, name, script)
	return err
}

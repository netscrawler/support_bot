package lua

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PluginRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *PluginRepository {
	return &PluginRepository{db: db}
}

type sctiptDBO struct {
	ID     int64  `db:"id"`
	Name   string `db:"name"`
	Script string `db:"script"`
}

func (r *PluginRepository) GetByName(ctx context.Context, name string) (string, error) {
	const query = `Select * from lua_scripts where name = $1`
	var plugin sctiptDBO
	err := r.db.GetContext(ctx, &plugin, query, name)
	if err != nil {
		return "", fmt.Errorf("script repo: get_by_name error: %w", err)
	}
	return plugin.Script, nil
}

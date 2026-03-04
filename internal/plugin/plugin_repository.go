package plugins

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type PluginRepository struct {
	db *sqlx.DB

	log *slog.Logger
}

func (mr *PluginRepository) LoadAllActive(ctx context.Context) ([]LuaPlugin, error) {
	return nil, nil
}

func (mr *PluginRepository) LoadByName(ctx context.Context) (LuaPlugin, error) {
	return LuaPlugin{}, nil
}

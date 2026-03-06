package plugins

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type LuaPluginDTO struct {
	Id          int    `db:"id"`
	Name        string `db:"name"`
	Version     string `db:"version"`
	Description string `db:"description"`
	Author      string `db:"author"`

	PluginStr string `db:"plugin_str"`

	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

type PluginRepository struct {
	db *sqlx.DB
}

func (pr *PluginRepository) LoadByName(ctx context.Context, name string) (LuaPluginDTO, error) {
	const op = "PluginRepository.LoadByName"
	if err := ctx.Err(); err != nil {
		return LuaPluginDTO{}, fmt.Errorf("%s: %w", op, err)
	}

	const stmt = `select id,
name, 
version, 
description,
author, 
plugin_str, 
created_at, updated_at
from plugins
where name = $1 order by created_at desc limit 1`

	var plug LuaPluginDTO

	err := pr.db.GetContext(ctx, &plug, stmt, name)
	if err != nil {
		return LuaPluginDTO{}, fmt.Errorf("%s: %w", op, err)
	}

	return plug, nil
}

func (pr *PluginRepository) LoadByNameAndVersion(ctx context.Context, name string, version string) (LuaPluginDTO, error) {
	const op = "PluginRepository.LoadByNameAndVersion"
	if err := ctx.Err(); err != nil {
		return LuaPluginDTO{}, fmt.Errorf("%s: %w", op, err)
	}

	if version == "" || version == "latest" {
		return pr.LoadByName(ctx, name)
	}

	const stmt = `select id, 
name, 
version, 
description, 
author, 
plugin_str, 
created_at, updated_at 
from plugins
where name = $1 and version = $2 limit 1`

	var plug LuaPluginDTO

	err := pr.db.GetContext(ctx, &plug, stmt, name, version)
	if err != nil {
		return LuaPluginDTO{}, fmt.Errorf("%s: %w", op, err)
	}

	return plug, nil
}

func (pr *PluginRepository) LoadByID(ctx context.Context, id int) (LuaPluginDTO, error) {
	const op = "PluginRepository.LoadByID"
	if err := ctx.Err(); err != nil {
		return LuaPluginDTO{}, fmt.Errorf("%s: %w", op, err)
	}

	const stmt = `select
id,
name,
version,
description,
author,
plugin_str,
created_at, updated_at
from plugins where id = $1`

	var plug LuaPluginDTO

	err := pr.db.GetContext(ctx, &plug, stmt, id)
	if err != nil {
		return LuaPluginDTO{}, fmt.Errorf("%s: %w", op, err)
	}

	return plug, nil
}

func (pr *PluginRepository) LoadByNameAll(ctx context.Context, name string) ([]LuaPluginDTO, error) {
	const op = "PluginRepository.LoadByNameAll"
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	const stmt = `select
id,
name,
version,
description,
author,
plugin_str,
created_at, updated_at
from plugins where name = $1 order by created_at desc`

	var plug []LuaPluginDTO

	err := pr.db.SelectContext(ctx, &plug, stmt, name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return plug, nil
}

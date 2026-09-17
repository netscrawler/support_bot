package store

import (
	"context"
	"fmt"
	"support_bot/internal/db/sqlcgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ScriptStore хранит Lua-скрипты (таблица lua_scripts) и объединяет
// два ранее раздельных доступа к ней: чтение (lua.PluginRepository.GetByName)
// и сохранение (repository.Script.Save).
type ScriptStore struct{ q *sqlcgen.Queries }

func NewScriptStore(pool *pgxpool.Pool) *ScriptStore {
	return &ScriptStore{q: sqlcgen.New(pool)}
}

// GetByName реализует lua.PluginProvider для загрузки исходного кода скрипта.
func (s *ScriptStore) GetByName(ctx context.Context, name string) (string, error) {
	script, err := s.q.GetScriptByName(ctx, name)
	if err != nil {
		return "", fmt.Errorf("get script by name: %w", translateNoRows(err))
	}

	return script, nil
}

// Save сохраняет скрипт, не перезаписывая существующий с тем же именем —
// CLI-команда script save исторически не обновляет уже сохранённые скрипты.
func (s *ScriptStore) Save(ctx context.Context, name, script string) error {
	if err := s.q.SaveScriptIfAbsent(
		ctx,
		sqlcgen.SaveScriptIfAbsentParams{Name: name, Script: script},
	); err != nil {
		return fmt.Errorf("save script: %w", err)
	}

	return nil
}

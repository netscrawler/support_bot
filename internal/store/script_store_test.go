package store

import (
	"context"
	"errors"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestScriptStore_GetByName_ReturnsScript проверяет, что GetByName выполняет
// GetScriptByName и возвращает исходный код Lua-скрипта без изменений.
func TestScriptStore_GetByName_ReturnsScript(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("select script from lua_scripts where name = \\$1").
		WithArgs("weekly.lua").
		WillReturnRows(pgxmock.NewRows([]string{"script"}).AddRow("return 1"))

	s := &ScriptStore{q: sqlcgen.New(pool)}
	got, err := s.GetByName(context.Background(), "weekly.lua")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}
	if got != "return 1" {
		t.Fatalf("GetByName() = %q, want %q", got, "return 1")
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestScriptStore_GetByName_NoRowsReturnsNotFound проверяет, что отсутствие
// строки транслируется в models.ErrNotFound на границе пакета store.
func TestScriptStore_GetByName_NoRowsReturnsNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("select script from lua_scripts where name = \\$1").
		WithArgs("missing.lua").
		WillReturnError(pgx.ErrNoRows)

	s := &ScriptStore{q: sqlcgen.New(pool)}
	_, err = s.GetByName(context.Background(), "missing.lua")
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("GetByName() error = %v, want wrapping models.ErrNotFound", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestScriptStore_Save_InsertsWithoutOverwritingOnConflict проверяет, что Save
// выполняет INSERT ... ON CONFLICT (name) DO NOTHING с переданными именем и
// текстом скрипта, не перезаписывая уже существующую запись.
func TestScriptStore_Save_InsertsWithoutOverwritingOnConflict(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectExec("insert into lua_scripts\\(name, script\\) values \\(\\$1, \\$2\\) on conflict \\(name\\) do nothing").
		WithArgs("weekly.lua", "return 1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := &ScriptStore{q: sqlcgen.New(pool)}
	if err := s.Save(context.Background(), "weekly.lua", "return 1"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

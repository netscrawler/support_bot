package store

import (
	"context"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// NewUserStoreForTest строит UserStore поверх уже созданного *sqlcgen.Queries
// (например, на основе pgxmock-пула) для модульных тестов. Продакшен-код
// всегда использует NewUserStore.
func NewUserStoreForTest(q *sqlcgen.Queries) *UserStore {
	return &UserStore{q: q, log: slog.Default()}
}

// TestUserStore_GetByUsername_ReturnsPopulatedUser проверяет, что
// GetByUsername возвращает реально заполненную структуру, а не nil —
// это баг, который был в старой реализации (var user *models.User без
// выделения памяти, переданный в db.GetContext, приводил к панике при
// любом совпадении).
func TestUserStore_GetByUsername_ReturnsPopulatedUser(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	username := "ivan"
	firstName := "Ivan"

	pool.ExpectQuery("select id, telegram_id, username, first_name, last_name, role").
		WithArgs(&username).
		WillReturnRows(pgxmock.NewRows(
			[]string{"id", "telegram_id", "username", "first_name", "last_name", "role"},
		).AddRow(int32(1), int64(42), &username, &firstName, (*string)(nil), sqlcgen.UserRoleAdmin))

	s := NewUserStoreForTest(sqlcgen.New(pool))

	user, err := s.GetByUsername(context.Background(), username)
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if user == nil {
		t.Fatal("GetByUsername() returned nil user, want a populated *models.User")
	}
	if user.TelegramID != 42 || user.Username != username || user.Role != "admin" {
		t.Errorf("GetByUsername() = %+v, want TelegramID=42 Username=%q Role=admin", user, username)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestUserStore_GetAllAdmins_ReturnsOnlyAdminAndPrimaryRoles проверяет,
// что GetAllAdmins возвращает только пользователей с ролью admin/primary —
// именно так фильтровала старая реализация (role in ('admin', 'primary')).
func TestUserStore_GetAllAdmins_ReturnsOnlyAdminAndPrimaryRoles(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	adminName := "admin1"

	pool.ExpectQuery("select id, telegram_id, username, first_name, last_name, role\nfrom users\nwhere role in").
		WillReturnRows(pgxmock.NewRows(
			[]string{"id", "telegram_id", "username", "first_name", "last_name", "role"},
		).AddRow(int32(3), int64(99), &adminName, (*string)(nil), (*string)(nil), sqlcgen.UserRoleAdmin))

	s := NewUserStoreForTest(sqlcgen.New(pool))

	admins, err := s.GetAllAdmins(context.Background())
	if err != nil {
		t.Fatalf("GetAllAdmins() error = %v", err)
	}
	if len(admins) != 1 || admins[0].Role != "admin" {
		t.Errorf("GetAllAdmins() = %+v, want exactly 1 admin", admins)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

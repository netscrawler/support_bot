package store

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// NewChatStoreForTest строит ChatStore поверх уже созданного *sqlcgen.Queries
// (например, на основе pgxmock-пула) для модульных тестов. Продакшен-код
// всегда использует NewChatStore.
func NewChatStoreForTest(q *sqlcgen.Queries) *ChatStore {
	return &ChatStore{q: q, log: slog.Default()}
}

// TestChatStore_GetByTitle_ReturnsPopulatedChat проверяет, что GetByTitle
// возвращает реально заполненную структуру, а не nil-указатель — это баг,
// который был в старой реализации (var chat *models.TgChatDTO без выделения
// памяти, переданный в db.GetContext, приводил к панике при любом совпадении).
func TestChatStore_GetByTitle_ReturnsPopulatedChat(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	title := "Test Chat"
	description := "some description"

	pool.ExpectQuery("select id, chat_id, title, type, description, is_active, ch_type").
		WithArgs(&title).
		WillReturnRows(pgxmock.NewRows(
			[]string{"id", "chat_id", "title", "type", "description", "is_active", "ch_type"},
		).AddRow(int32(1), int64(555), &title, "group", &description, true, "tg"))

	s := NewChatStoreForTest(sqlcgen.New(pool))

	chat, err := s.GetByTitle(context.Background(), title)
	if err != nil {
		t.Fatalf("GetByTitle() error = %v", err)
	}
	if chat == nil {
		t.Fatal("GetByTitle() returned nil chat, want a populated *models.TgChatDTO")
	}
	if chat.ChatID != 555 || chat.Title != title || chat.Type != "group" {
		t.Errorf("GetByTitle() = %+v, want ChatID=555 Title=%q Type=group", chat, title)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestChatStore_Exists_ReturnsTrueForKnownChatID проверяет, что Exists
// возвращает true, когда запрос находит строку по chat_id.
func TestChatStore_Exists_ReturnsTrueForKnownChatID(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("select id from chats where chat_id = \\$1").
		WithArgs(int64(555)).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(1)))

	s := NewChatStoreForTest(sqlcgen.New(pool))

	exists, err := s.Exists(context.Background(), 555)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestChatStore_Exists_NoRowsIsFalse проверяет, что отсутствие строки
// (pgx.ErrNoRows) транслируется в exists=false без ошибки, а не в
// models.ErrNotFound — вызывающему коду достаточно булева результата.
func TestChatStore_Exists_NoRowsIsFalse(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("select id from chats where chat_id = \\$1").
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	s := NewChatStoreForTest(sqlcgen.New(pool))

	exists, err := s.Exists(context.Background(), 999)
	if err != nil {
		t.Fatalf("Exists() error = %v, want nil", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestChatStore_Exists_PropagatesQueryError проверяет, что ошибка БД,
// отличная от pgx.ErrNoRows, возвращается вызывающему коду обёрнутой,
// но с сохранением исходного значения для errors.Is.
func TestChatStore_Exists_PropagatesQueryError(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	dbErr := errors.New("connection reset")

	pool.ExpectQuery("select id from chats where chat_id = \\$1").
		WithArgs(int64(100)).
		WillReturnError(dbErr)

	s := NewChatStoreForTest(sqlcgen.New(pool))

	exists, err := s.Exists(context.Background(), 100)
	if exists {
		t.Error("Exists() = true, want false")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("Exists() error = %v, want wrapped %v", err, dbErr)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestChatStore_GetAll_FiltersInactiveChats проверяет, что GetAll возвращает
// именно чаты с is_active = false — так и было в старой реализации; это
// сознательно сохранённое поведение, а не то, что предполагает имя метода.
func TestChatStore_GetAll_FiltersInactiveChats(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	title := "Inactive Chat"

	pool.ExpectQuery("select id, chat_id, title, type, description, is_active, ch_type\nfrom chats\nwhere is_active = false").
		WillReturnRows(pgxmock.NewRows(
			[]string{"id", "chat_id", "title", "type", "description", "is_active", "ch_type"},
		).AddRow(int32(2), int64(777), &title, "private", (*string)(nil), false, "tg"))

	s := NewChatStoreForTest(sqlcgen.New(pool))

	chats, err := s.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(chats) != 1 || chats[0].IsActive {
		t.Errorf("GetAll() = %+v, want exactly 1 chat with IsActive=false", chats)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

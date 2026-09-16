package store

import (
	"context"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"testing"

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

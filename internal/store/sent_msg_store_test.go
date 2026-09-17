package store

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

// NewSentMsgStoreForTest строит SentMsgStore поверх уже созданного
// *sqlcgen.Queries для тестов, которым не нужна ExecTx/BeginTx-машинерия
// напрямую через поле pool. Продакшен-код всегда использует NewSentMsgStore.
func NewSentMsgStoreForTest(q *sqlcgen.Queries) *SentMsgStore {
	return &SentMsgStore{q: q, log: slog.Default()}
}

// TestSentMsgStore_WithLockedMsgsToDelete_BeginFails проверяет, что fn не
// вызывается, если открыть транзакцию не удалось, и ошибка Begin
// пробрасывается наружу без изменений.
func TestSentMsgStore_WithLockedMsgsToDelete_BeginFails(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	beginErr := errors.New("connection pool exhausted")
	pool.ExpectBegin().WillReturnError(beginErr)

	s := NewSentMsgStoreForTest(sqlcgen.New(pool))

	called := false
	err = s.withLockedMsgsToDeleteWithPool(
		context.Background(),
		pool,
		func(context.Context, models.SentMessage) bool {
			called = true
			return true
		},
	)

	if called {
		t.Fatal("fn was called despite Begin failing")
	}
	if !errors.Is(err, beginErr) {
		t.Fatalf("WithLockedMsgsToDelete() error = %v, want wrapping %v", err, beginErr)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSentMsgStore_WithLockedMsgsToDelete_CallbackDecidesMarkDeleted проверяет,
// что решение о пометке "удалено" принимает переданный fn для каждой строки
// независимо: из двух загруженных сообщений помечается только то, для
// которого fn вернул true.
func TestSentMsgStore_WithLockedMsgsToDelete_CallbackDecidesMarkDeleted(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	title := "Report Chat"
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	pool.ExpectBegin()
	pool.ExpectQuery("select id, chat_id, thread_id, message_id, message_id_str, title, sent_at, deleted, ch_type").
		WillReturnRows(pgxmock.NewRows(
			[]string{
				"id", "chat_id", "thread_id", "message_id",
				"message_id_str", "title", "sent_at", "deleted", "ch_type",
			},
		).
			AddRow(int32(1), int64(100), int32(0), int64(11), (*string)(nil), title, now, false, "tg").
			AddRow(int32(2), int64(200), int32(0), int64(22), (*string)(nil), title, now, false, "tg"),
		)
	pool.ExpectExec("update sent_messages set deleted = true").
		WithArgs(int32(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	pool.ExpectCommit()
	pool.ExpectRollback() // no-op после успешного коммита, как и в exec_tx_test.go

	s := NewSentMsgStoreForTest(sqlcgen.New(pool))

	var seenIDs []int64
	err = s.withLockedMsgsToDeleteWithPool(
		context.Background(),
		pool,
		func(_ context.Context, m models.SentMessage) bool {
			seenIDs = append(seenIDs, m.ID)
			return m.ID == 1 // помечаем удалённым только первое сообщение
		},
	)
	if err != nil {
		t.Fatalf("WithLockedMsgsToDelete() error = %v", err)
	}
	if len(seenIDs) != 2 {
		t.Fatalf("fn called %d times, want 2", len(seenIDs))
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"unmet expectations (MarkSentMsgDeleted must be called exactly once, for id=1): %v",
			err,
		)
	}
}

// TestSentMsgStore_RemoveDeletedMessages_ReturnsRowsAffected проверяет, что
// метод возвращает реальное количество удалённых строк, а не только nil-ошибку.
func TestSentMsgStore_RemoveDeletedMessages_ReturnsRowsAffected(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	pool.ExpectExec("delete from sent_messages where deleted = true").
		WillReturnResult(pgxmock.NewResult("DELETE", 3))
	pool.ExpectCommit()
	pool.ExpectRollback()

	s := NewSentMsgStoreForTest(sqlcgen.New(pool))

	removed, err := s.removeDeletedMessagesWithPool(context.Background(), pool)
	if err != nil {
		t.Fatalf("RemoveDeletedMessages() error = %v", err)
	}
	if removed != 3 {
		t.Errorf("RemoveDeletedMessages() = %d, want 3", removed)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSentMsgStore_MarkEndOfDayMsgDeleted_BeginTxFails переносит два кейса из
// прежнего internal/orchestrator/result_repo_test.go: если BeginTx не
// удался, метод должен выполнить запрос напрямую без транзакции, а не
// упасть с паникой на nil tx.
func TestSentMsgStore_MarkEndOfDayMsgDeleted_BeginTxFails(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted}).
		WillReturnError(errors.New("connection pool exhausted"))
	pool.ExpectExec("UPDATE sent_messages").WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	s := NewSentMsgStoreForTest(sqlcgen.New(pool))

	err = s.markEndOfDayMsgDeletedWithPool(context.Background(), pool)
	if err != nil {
		t.Fatalf("MarkEndOfDayMsgDeleted() error = %v, want nil (fallback exec succeeded)", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSentMsgStore_MarkEndOfDayMsgDeleted_BeginTxFails_FallbackErrorPropagates
// проверяет вторую половину того же старого теста: если и запасной запрос
// без транзакции тоже падает, эта ошибка должна дойти до вызывающего кода.
func TestSentMsgStore_MarkEndOfDayMsgDeleted_BeginTxFails_FallbackErrorPropagates(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	fallbackErr := errors.New("exec failed")
	pool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted}).
		WillReturnError(errors.New("begin failed"))
	pool.ExpectExec("UPDATE sent_messages").WillReturnError(fallbackErr)

	s := NewSentMsgStoreForTest(sqlcgen.New(pool))

	err = s.markEndOfDayMsgDeletedWithPool(context.Background(), pool)
	if !errors.Is(err, fallbackErr) {
		t.Fatalf("MarkEndOfDayMsgDeleted() error = %v, want wrapping %v", err, fallbackErr)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

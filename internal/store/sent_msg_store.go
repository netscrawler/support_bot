package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SentMsgStore реализует orchestrator.SentMsgSaver и хранилище для
// internal/orchestrator.Deleter поверх sqlc-запросов sent_messages.sql.
// Заменяет internal/orchestrator.SentMsgRepository.
type SentMsgStore struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
	log  *slog.Logger
}

// txOptsBeginner — минимальный контракт для методов, которым нужен явный
// уровень изоляции транзакции (в отличие от store.ExecTx, который всегда
// использует уровень по умолчанию). Отдельный от txBeginner
// (internal/store/exec_tx.go), т.к. остальным Store такой контроль не нужен.
type txOptsBeginner interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

func NewSentMsgStore(pool *pgxpool.Pool, log *slog.Logger) *SentMsgStore {
	return &SentMsgStore{
		pool: pool,
		q:    sqlcgen.New(pool),
		log:  log.With(slog.Any("module", "store.sent_msg")),
	}
}

// SaveTgMsg сохраняет отправленные сообщения по одному, как и старая
// реализация (без единой транзакции на всю пачку): ошибка по одному
// сообщению не должна мешать сохранить остальные, поэтому ошибки
// накапливаются через errors.Join и возвращаются все сразу.
func (s *SentMsgStore) SaveTgMsg(
	ctx context.Context,
	reportName string,
	msgs []models.SentMessage,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("save sent message: %w", err)
	}

	var saveErr error

	for _, msg := range msgs {
		err := s.q.InsertSentMessage(ctx, sqlcgen.InsertSentMessageParams{
			ChatID:       msg.ChatID,
			ThreadID:     int32(msg.ThreadID), //nolint:gosec // telegram/max thread ids fit int32
			MessageID:    int64(msg.MessageID),
			MessageIDStr: msg.MessageIDStr,
			Title:        msg.Title,
			SentAt:       pgtype.Timestamptz{Time: msg.Time, Valid: true},
			ReportName:   reportName,
			ChType:       msg.ChType,
		})
		if err != nil {
			saveErr = errors.Join(saveErr, err)
		}
	}

	return saveErr
}

// WithLockedMsgsToDelete блокирует ещё не удалённые сообщения (select ... for
// update skip locked) в рамках одной транзакции и вызывает fn для каждого —
// решение о том, помечать ли сообщение удалённым, остаётся за вызывающим
// кодом (Deleter), поскольку это его бизнес-логика выбора адаптера доставки
// и обработки ошибки удаления, а не хранилища. Раньше эта блокировка
// держалась через internal/pkg/uow, который отдавал наружу *sqlx.Tx —
// pgx.Tx, полученный внутри ExecTx, так вынести из замыкания нельзя, отсюда
// колбэк вместо возврата (msgs, tx).
func (s *SentMsgStore) WithLockedMsgsToDelete(
	ctx context.Context,
	fn func(ctx context.Context, msg models.SentMessage) (markDeleted bool),
) error {
	return s.withLockedMsgsToDeleteWithPool(ctx, s.pool, fn)
}

// RemoveDeletedMessages физически удаляет сообщения, уже помеченные
// deleted = true, и возвращает количество удалённых строк. ReadCommitted —
// как и в старой реализации: между чтением и записью здесь нет решения,
// зависящего от промежуточного состояния БД, более строгая изоляция не нужна.
func (s *SentMsgStore) RemoveDeletedMessages(ctx context.Context) (int64, error) {
	return s.removeDeletedMessagesWithPool(ctx, s.pool)
}

// MarkEndOfDayMsgDeleted помечает удалённым (без физического удаления)
// последнее за вчерашний день сообщение на пару report_name+chat_id — оно
// остаётся видимым в чате как финальный отчёт за день. Если открыть
// транзакцию не удалось, запрос выполняется без неё — то же поведение, что
// было в internal/orchestrator.SentMsgRepository.markEndOfDayMsgDeleted;
// без этого запасного пути ежедневная очистка могла не произойти вовсе при
// временной недоступности начала транзакции.
func (s *SentMsgStore) MarkEndOfDayMsgDeleted(ctx context.Context) error {
	return s.markEndOfDayMsgDeletedWithPool(ctx, s.pool)
}

func (s *SentMsgStore) withLockedMsgsToDeleteWithPool(
	ctx context.Context,
	pool txBeginner,
	fn func(ctx context.Context, msg models.SentMessage) (markDeleted bool),
) error {
	return ExecTx(ctx, pool, func(q *sqlcgen.Queries) error {
		rows, err := q.LoadSentMsgsToDeleteForUpdate(ctx)
		if err != nil {
			return fmt.Errorf("load messages to delete: %w", err)
		}

		for _, row := range rows {
			msg := mapSentMsgToDeleteRow(row)

			if !fn(ctx, msg) {
				continue
			}

			// Ошибка здесь возвращается, а не логируется: она обрывает всю
			// ExecTx и приводит к rollback — что произошло бы в реальном
			// Postgres в любом случае, т.к. ошибка внутри транзакции делает
			// недействительными все последующие операторы вплоть до commit.
			// Возврат ошибки раньше просто не даёт выполнить заведомо
			// обречённые запросы для оставшихся строк. Вызывающий код
			// (Deleter.delete) сам логирует итоговую ошибку.
			//nolint:gosec // ids fit int32, see schema
			if err := q.MarkSentMsgDeleted(ctx, int32(msg.ID)); err != nil {
				return fmt.Errorf("mark deleted id=%d: %w", msg.ID, err)
			}
		}

		return nil
	})
}

func (s *SentMsgStore) removeDeletedMessagesWithPool(
	ctx context.Context,
	pool txOptsBeginner,
) (int64, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	removed, err := sqlcgen.New(tx).DeleteAllMarkedSentMessages(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete marked messages: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return removed, nil
}

func (s *SentMsgStore) markEndOfDayMsgDeletedWithPool(
	ctx context.Context,
	pool txOptsBeginner,
) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		s.log.ErrorContext(ctx, "begin tx failed, continue without tx", slog.Any("error", err))

		return s.q.MarkEndOfDayMsgDeleted(ctx)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := sqlcgen.New(tx).MarkEndOfDayMsgDeleted(ctx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func mapSentMsgToDeleteRow(row sqlcgen.LoadSentMsgsToDeleteForUpdateRow) models.SentMessage {
	return models.SentMessage{
		ID:           int64(row.ID),
		ChatID:       row.ChatID,
		ThreadID:     int(row.ThreadID),
		MessageID:    int(row.MessageID),
		MessageIDStr: row.MessageIDStr,
		Title:        row.Title,
		Time:         row.SentAt.Time,
		Deleted:      row.Deleted,
		ChType:       row.ChType,
	}
}

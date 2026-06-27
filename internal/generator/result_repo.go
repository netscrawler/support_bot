package generator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	models "support_bot/internal/models/report"
	"support_bot/internal/pkg/uow"

	"github.com/jmoiron/sqlx"
)

type SentMsgRepository struct {
	db *sqlx.DB

	log *slog.Logger
}

func NewResultRepository(db *sqlx.DB, log *slog.Logger) *SentMsgRepository {
	return &SentMsgRepository{
		db:  db,
		log: log,
	}
}

func (rr *SentMsgRepository) SaveTgMsg(ctx context.Context, reportName string, msgs []models.TgMessage) error {
	const query = `insert into sent_messages(chat_id, thread_id, message_id, title, sent_at, report_name) values ($1, $2, $3, $4, $5, $6);`

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("chat repository create: %w", err)
	}

	var saveErr error

	for _, msg := range msgs {
		_, err := rr.db.ExecContext(
			ctx,
			query,
			msg.ChatID,
			msg.ThreadID,
			msg.MessageID,
			msg.Title,
			msg.Time,
			reportName,
		)
		if err != nil {
			saveErr = errors.Join(saveErr, err)
		}
	}

	return saveErr
}

func (rr *SentMsgRepository) LoadMsgToDelete(ctx context.Context) ([]models.TgMessage, uow.UOW, error) {
	const query = `select id, chat_id, thread_id, message_id, title, sent_at, deleted from sent_messages where deleted = False AND sent_at >= CURRENT_DATE - INTERVAL '1 day'
	AND sent_at < CURRENT_DATE for update skip locked;`

	var msgs []models.TgMessage

	tx, err := rr.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}

	if err := tx.SelectContext(ctx, &msgs, query); err != nil {
		return nil, nil, err
	}

	u := uow.NewUOW(tx)

	return msgs, u, nil
}

func (rr *SentMsgRepository) MarkDeleted(ctx context.Context, id int64, u uow.UOW) error {
	const query = `update sent_messages set deleted = true where id = $1;`

	if _, err := u.ExecContext(ctx, query, id); err != nil {
		return err
	}

	return nil
}

func (rr *SentMsgRepository) RemoveDeletedMessages(ctx context.Context) (int64, error) {
	const query = `delete from sent_messages where deleted = true;`

	tx, err := rr.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete all messages from db: %w", err)
	}

	removed, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}
	return removed, tx.Commit()
}

func (rr *SentMsgRepository) MarkEndOfDayMsgDeleted(ctx context.Context) error {
	const query = `UPDATE sent_messages sm
SET deleted = TRUE
FROM (
    SELECT DISTINCT ON (report_name, chat_id)
        id
    FROM sent_messages
    WHERE deleted = FALSE
      AND sent_at >= CURRENT_DATE - INTERVAL '1 day'
      AND sent_at < CURRENT_DATE
    ORDER BY report_name, chat_id, sent_at DESC, id DESC
) last_msgs
WHERE sm.id = last_msgs.id;`

	tx, err := rr.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		rr.log.ErrorContext(ctx, "begin tx failed, continue without tx", err)
		_, err = rr.db.ExecContext(ctx, query)
		if err != nil {
			return err
		}
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return tx.Commit()
}

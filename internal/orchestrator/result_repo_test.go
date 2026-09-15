package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// markEndOfDayMsgDeleted must not touch tx once BeginTxx has failed: falling
// through to defer tx.Rollback()/tx.ExecContext() on a nil tx panics.
func TestSentMsgRepository_MarkEndOfDayMsgDeleted_BeginTxFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rr := NewResultRepository(sqlx.NewDb(db, "sqlmock"), slog.New(slog.DiscardHandler))

	mock.ExpectBegin().WillReturnError(errors.New("connection pool exhausted"))
	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))

	err = rr.markEndOfDayMsgDeleted(context.Background())

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSentMsgRepository_MarkEndOfDayMsgDeleted_BeginTxFails_FallbackErrorPropagates(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rr := NewResultRepository(sqlx.NewDb(db, "sqlmock"), slog.New(slog.DiscardHandler))

	fallbackErr := errors.New("exec failed")
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))
	mock.ExpectExec(".*").WillReturnError(fallbackErr)

	err = rr.markEndOfDayMsgDeleted(context.Background())

	require.ErrorIs(t, err, fallbackErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

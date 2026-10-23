package postgres

import (
	"context"
	"support_bot/internal/pkg/secret"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestPoolConfig(t *testing.T) {
	f := false
	dsn := secret.String(
		"postgres://u:p@localhost:5432/db",
	) //nolint:gosec // фиктивный DSN для теста

	pc, err := poolConfig(Config{DSN: dsn})
	require.NoError(t, err)
	require.Equal(t, "on", pc.ConnConfig.RuntimeParams["default_transaction_read_only"])
	require.EqualValues(t, 4, pc.MaxConns)

	pc, err = poolConfig(Config{DSN: dsn, ReadOnly: &f, MaxConns: 7})
	require.NoError(t, err)
	require.NotContains(t, pc.ConnConfig.RuntimeParams, "default_transaction_read_only")
	require.EqualValues(t, 7, pc.MaxConns)

	require.Equal(t, 30*time.Second, Config{}.withDefaults().StatementTimeout)

	_, err = poolConfig(Config{DSN: "://bad"})
	require.Error(t, err)
}

func newMocked(t *testing.T) (*Collector, pgxmock.PgxPoolIface) {
	t.Helper()

	m, err := pgxmock.NewPool()
	require.NoError(t, err)

	return &Collector{dbs: map[string]target{"main": {pool: m, timeout: 30 * time.Second}}}, m
}

func TestFetch(t *testing.T) {
	c, m := newMocked(t)
	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	m.ExpectBeginTx(pgx.TxOptions{AccessMode: pgx.ReadOnly})
	m.ExpectExec(`SET LOCAL statement_timeout = 30000`).
		WillReturnResult(pgxmock.NewResult("SET", 0))
	m.ExpectQuery(`select`).WithArgs(pgx.NamedArgs{"a": "1"}).
		WillReturnRows(pgxmock.NewRows([]string{"n", "t"}).AddRow(int64(5), ts))
	m.ExpectRollback()

	got, err := c.Fetch(context.Background(), "main", "select @a", map[string]string{"a": "1"})
	require.NoError(t, err)
	require.Equal(t, []map[string]any{{"n": float64(5), "t": "2024-01-02T03:04:05Z"}}, got)
	require.NoError(t, m.ExpectationsWereMet()) // Commit не ожидался, значит и не вызывался
}

func TestFetchEmpty(t *testing.T) {
	c, m := newMocked(t)

	m.ExpectBeginTx(pgx.TxOptions{AccessMode: pgx.ReadOnly})
	m.ExpectExec(`SET LOCAL`).WillReturnResult(pgxmock.NewResult("SET", 0))
	m.ExpectQuery(`select`).WithArgs(pgx.NamedArgs{}).WillReturnRows(pgxmock.NewRows([]string{"n"}))
	m.ExpectRollback()

	got, err := c.Fetch(context.Background(), "main", "select 1 where false", nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
	require.NoError(t, m.ExpectationsWereMet())
}

func TestFetchQueryError(t *testing.T) {
	c, m := newMocked(t)

	m.ExpectBeginTx(pgx.TxOptions{AccessMode: pgx.ReadOnly})
	m.ExpectExec(`SET LOCAL`).WillReturnResult(pgxmock.NewResult("SET", 0))
	m.ExpectQuery(`select`).WithArgs(pgx.NamedArgs{}).WillReturnError(context.DeadlineExceeded)
	m.ExpectRollback()

	_, err := c.Fetch(context.Background(), "main", "select 1", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, m.ExpectationsWereMet())
}

func TestFetchUnknownDB(t *testing.T) {
	c, _ := newMocked(t)

	_, err := c.Fetch(context.Background(), "nope", "select 1", nil)
	require.EqualError(t, err, `unknown database "nope"`)
}

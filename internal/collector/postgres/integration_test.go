package postgres

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"support_bot/internal/pkg/secret"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestIntegrationReadOnly(t *testing.T) {
	dsn := os.Getenv("PG_COLLECTOR_TEST_DSN")
	if dsn == "" {
		t.Skip("PG_COLLECTOR_TEST_DSN не задан")
	}

	ctx := context.Background()

	admin, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = admin.Exec(ctx, "drop table if exists pgcoll_tmp")
		_ = admin.Close(ctx)
	})

	_, err = admin.Exec(
		ctx,
		"drop table if exists pgcoll_tmp; create table pgcoll_tmp(i int); insert into pgcoll_tmp values (1)",
	)
	require.NoError(t, err)

	c, err := New(
		ctx,
		map[string]Config{
			"db": {DSN: secret.String(dsn), StatementTimeout: time.Second},
		},
		slog.New(slog.DiscardHandler),
	)
	require.NoError(t, err)
	t.Cleanup(c.Close)

	_, err = c.Fetch(ctx, "db", "delete from pgcoll_tmp", nil)
	require.Error(t, err)

	_, err = c.Fetch(ctx, "db", "create table pgcoll_x(i int)", nil)
	require.Error(t, err)

	_, err = c.Fetch(ctx, "db", "select 1; delete from pgcoll_tmp", nil)
	require.Error(t, err)

	var n int

	require.NoError(
		t,
		admin.QueryRow(ctx, "select count(*) from pgcoll_tmp").Scan(&n),
	)
	require.Equal(t, 1, n)

	start := time.Now()

	_, err = c.Fetch(ctx, "db", "select pg_sleep(5)", nil)
	require.Error(t, err)
	require.Less(t, time.Since(start), 4*time.Second)

	got, err := c.Fetch(ctx, "db", "show default_transaction_read_only", nil)
	require.NoError(t, err)
	require.Equal(
		t,
		[]map[string]any{{"default_transaction_read_only": "on"}},
		got,
	)

	got, err = c.Fetch(
		ctx,
		"db",
		"select @v::text as v",
		map[string]string{"v": "1; drop table x"},
	)
	require.NoError(t, err)
	require.Equal(t, []map[string]any{{"v": "1; drop table x"}}, got)
}

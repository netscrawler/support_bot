package metrics

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRegisterPool(t *testing.T) {
	// pgxpool.New ленив: к БД не подключается до первого Acquire.
	pool, err := pgxpool.New(
		context.Background(),
		"postgres://u:p@127.0.0.1:1/db",
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	RegisterPool(pool.Stat)
	RegisterPool(pool.Stat) // повторная регистрация не паникует

	_, body := get(t, func(context.Context) error { return nil }, "/metrics")
	for _, name := range []string{"sbot_db_pool_acquired_conns", "sbot_db_pool_max_conns"} {
		if !strings.Contains(body, name) {
			t.Errorf("%s missing from /metrics", name)
		}
	}
}

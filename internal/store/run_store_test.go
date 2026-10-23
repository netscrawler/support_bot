package store

import (
	"context"
	"strings"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func newRunStoreForTest(t *testing.T) (*RunStore, pgxmock.PgxPoolIface) {
	t.Helper()

	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	t.Cleanup(pool.Close)

	return &RunStore{q: sqlcgen.New(pool)}, pool
}

func TestRunStore_Insert_TruncatesErrorByRunes(t *testing.T) {
	s, pool := newRunStoreForTest(t)

	long := strings.Repeat("ы", 5000)
	want := strings.Repeat("ы", 4000)
	now := time.Now()

	pool.ExpectExec("insert into report_runs").
		WithArgs("r", "cron", "failed", (*string)(nil), &want, (*int64)(nil), pgxmock.AnyArg(), pgxmock.AnyArg(), (*string)(nil)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := s.Insert(context.Background(), models.Run{
		ReportName: "r", Kind: models.RunKindCron, Status: models.RunStatusFailed,
		Error: long, StartedAt: now, FinishedAt: now,
	})
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunStore_Insert_EmptyStringsAreNil(t *testing.T) {
	s, pool := newRunStoreForTest(t)
	now := time.Now()

	pool.ExpectExec("insert into report_runs").
		WithArgs("r", "manual", "success", (*string)(nil), (*string)(nil), (*int64)(nil), pgxmock.AnyArg(), pgxmock.AnyArg(), (*string)(nil)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := s.Insert(context.Background(), models.Run{
		ReportName: "r", Kind: models.RunKindManual, Status: models.RunStatusSuccess,
		StartedAt: now, FinishedAt: now,
	})
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunStore_LastStatus(t *testing.T) {
	s, pool := newRunStoreForTest(t)

	pool.ExpectQuery("select status from report_runs").
		WithArgs("none").
		WillReturnError(pgx.ErrNoRows)
	st, ok, err := s.LastStatus(context.Background(), "none")
	if err != nil || ok || st != "" {
		t.Fatalf("no rows: got (%q, %v, %v), want (\"\", false, nil)", st, ok, err)
	}

	pool.ExpectQuery("select status from report_runs").WithArgs("r").
		WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("partial"))
	st, ok, err = s.LastStatus(context.Background(), "r")
	if err != nil || !ok || st != models.RunStatusPartial {
		t.Fatalf("row: got (%q, %v, %v), want (partial, true, nil)", st, ok, err)
	}
}

func TestRunStore_LastSuccessHash(t *testing.T) {
	s, pool := newRunStoreForTest(t)
	h := "abc"

	tests := []struct {
		name   string
		rows   *pgxmock.Rows
		err    error
		want   string
		wantOK bool
	}{
		{"hash", pgxmock.NewRows([]string{"data_hash"}).AddRow(&h), nil, "abc", true},
		{"no rows", nil, pgx.ErrNoRows, "", false},
		{"null", pgxmock.NewRows([]string{"data_hash"}).AddRow((*string)(nil)), nil, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := pool.ExpectQuery("select data_hash from report_runs").WithArgs("r")
			if tt.err != nil {
				q.WillReturnError(tt.err)
			} else {
				q.WillReturnRows(tt.rows)
			}

			got, ok, err := s.LastSuccessHash(context.Background(), "r")
			if err != nil || ok != tt.wantOK || got != tt.want {
				t.Fatalf("got (%q, %v, %v), want (%q, %v, nil)", got, ok, err, tt.want, tt.wantOK)
			}
		})
	}
}

func TestRunStore_List(t *testing.T) {
	s, pool := newRunStoreForTest(t)
	started := time.Now().Truncate(time.Second)
	reason, chat := "no data", int64(42)

	pool.ExpectQuery("from report_runs").WithArgs("r", int32(5)).
		WillReturnRows(pgxmock.NewRows([]string{
			"id",
			"report_name",
			"kind",
			"status",
			"reason",
			"error",
			"chat_id",
			"started_at",
			"finished_at",
		}).AddRow(int64(1), "r", "lk", "skipped", &reason, (*string)(nil), &chat, started, started))

	runs, err := s.List(context.Background(), "r", 5)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("len = %d, want 1", len(runs))
	}
	got := runs[0]
	if got.Kind != models.RunKindLK || got.Status != models.RunStatusSkipped ||
		got.Reason != "no data" || got.Error != "" || got.ChatID == nil || *got.ChatID != 42 ||
		!got.StartedAt.Equal(started) {
		t.Fatalf("unexpected run: %+v", got)
	}
}

func TestRunStore_DeleteOlderThan(t *testing.T) {
	s, pool := newRunStoreForTest(t)

	pool.ExpectExec("delete from report_runs").WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 7))

	n, err := s.DeleteOlderThan(context.Background(), time.Now())
	if err != nil || n != 7 {
		t.Fatalf("got (%d, %v), want (7, nil)", n, err)
	}
}

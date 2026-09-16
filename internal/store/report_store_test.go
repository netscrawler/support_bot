package store

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestReportStore_GetByID_LoadsAllFields(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pipelineID := int64(42)

	pool.ExpectQuery("select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows(
			[]string{
				"id",
				"name",
				"title",
				"active",
				"access_from_lk",
				"pipeline_id",
				"evaluation",
			},
		).
			AddRow(int32(1), "r1", "Report One", true, true, &pipelineID, new("1 == 1")))

	pool.ExpectQuery("select q.card_uuid, q.title, q.q_type, q.params").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows(
			[]string{"card_uuid", "title", "q_type", "params"},
		).AddRow("uuid-1", "Card One", "mb", []byte(`{"k":"v"}`)))

	pool.ExpectQuery("select\n    rc.name").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{
			"name",
			"config",
			"remote_path",
			"thread_id",
			"email_id",
			"type",
			"need_delete_after_end_of_day",
			"dest",
			"copy",
			"subject",
			"body",
			"chat_id",
			"chat_title",
			"chat_type",
			"chat_description",
			"chat_is_active",
			"chat_ch_type",
		}))

	pool.ExpectQuery("select ef.format, re.file_name").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{
			"format",
			"file_name",
			"template_id",
			"template_title",
			"template_type",
			"template_text",
			"sort_order",
		}))

	pool.ExpectQuery("select c.cron, c.name, c.description, c.is_active, c.event_type").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{
			"cron", "name", "description", "is_active", "event_type",
		}).AddRow("* * * * *", "every-minute", new("desc"), true, int32(0)))

	pool.ExpectQuery("select pipeline from pipelines").
		WithArgs(pipelineID).
		WillReturnRows(pgxmock.NewRows([]string{"pipeline"}).
			AddRow([]byte(`{"name":"p1","steps":[]}`)))

	s := NewReportStoreForTest(sqlcgen.New(pool))

	report, err := s.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if len(report.Crons) != 1 {
		t.Errorf(
			"Crons = %v, want 1 entry (this was silently dropped by the orchestrator's old loader)",
			report.Crons,
		)
	}
	if report.Pipeline == nil || report.Pipeline.Name != "p1" {
		t.Errorf(
			"Pipeline = %v, want non-nil with Name=p1 (this was silently dropped by the orchestrator's old loader)",
			report.Pipeline,
		)
	}
	if len(report.Queries) != 1 || report.Queries[0].Type != "mb" {
		t.Errorf(
			"Queries[0].Type = %q, want %q (this field was dropped by the tg_bot loader's card row type)",
			report.Queries[0].Type,
			"mb",
		)
	}
	if len(report.Queries) != 1 || len(report.Queries[0].RawParams) == 0 {
		t.Errorf(
			"Queries[0].RawParams empty, want %s (also dropped by the tg_bot loader)",
			`{"k":"v"}`,
		)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportStore_GetByID_PipelineNotFound covers the assemble() gap where
// GetPipelineByID's error bypassed translateNoRows: a dangling pipeline_id
// (pgx.ErrNoRows from the pipeline lookup) must surface as models.ErrNotFound
// like every other single-row lookup in this file, not a bare wrapped error.
func TestReportStore_GetByID_PipelineNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pipelineID := int64(42)

	pool.ExpectQuery("select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows(
			[]string{
				"id",
				"name",
				"title",
				"active",
				"access_from_lk",
				"pipeline_id",
				"evaluation",
			},
		).
			AddRow(int32(1), "r1", "Report One", true, true, &pipelineID, new("1 == 1")))

	pool.ExpectQuery("select q.card_uuid, q.title, q.q_type, q.params").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows(
			[]string{"card_uuid", "title", "q_type", "params"},
		))

	pool.ExpectQuery("select\n    rc.name").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{
			"name",
			"config",
			"remote_path",
			"thread_id",
			"email_id",
			"type",
			"need_delete_after_end_of_day",
			"dest",
			"copy",
			"subject",
			"body",
			"chat_id",
			"chat_title",
			"chat_type",
			"chat_description",
			"chat_is_active",
			"chat_ch_type",
		}))

	pool.ExpectQuery("select ef.format, re.file_name").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{
			"format",
			"file_name",
			"template_id",
			"template_title",
			"template_type",
			"template_text",
			"sort_order",
		}))

	pool.ExpectQuery("select c.cron, c.name, c.description, c.is_active, c.event_type").
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{
			"cron", "name", "description", "is_active", "event_type",
		}))

	pool.ExpectQuery("select pipeline from pipelines").
		WithArgs(pipelineID).
		WillReturnError(pgx.ErrNoRows)

	s := NewReportStoreForTest(sqlcgen.New(pool))

	_, err = s.GetByID(context.Background(), 1)
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want errors.Is(err, models.ErrNotFound)", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestReportStore_Create_AlreadyExists(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectBegin()
	pool.ExpectQuery("select exists").
		WithArgs("dup-report").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	pool.ExpectRollback()

	s := NewReportStore(nil, slog.New(slog.DiscardHandler))

	_, err = execTxOnMock(t, pool, s, models.Report{Name: "dup-report"})
	if !errors.Is(err, models.ErrAlreadyExist) {
		t.Fatalf("Create() error = %v, want models.ErrAlreadyExist", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// execTxOnMock runs Create against a store whose pool is a pgxmock pool,
// substituting ExecTx's underlying Begin call for the mock's.
func execTxOnMock(
	t *testing.T,
	pool pgxmock.PgxPoolIface,
	s *ReportStore,
	rep models.Report,
) (int64, error) {
	t.Helper()
	return s.createWithPool(context.Background(), pool, rep)
}

func TestReportStore_Create_NewReport_NoDependencies(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectBegin()
	pool.ExpectQuery("select exists").
		WithArgs("new-report").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	pool.ExpectQuery("select id from evaluate").
		WithArgs("1 == 1").
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("insert into evaluate").
		WithArgs("1 == 1").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(10)))
	pool.ExpectQuery("insert into reports").
		WithArgs("new-report", "New Report", int64(10), (*int64)(nil), false, true).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(100)))
	pool.ExpectCommit()

	id, err := execTxOnMock(
		t,
		pool,
		NewReportStore(nil, slog.New(slog.DiscardHandler)),
		models.Report{
			Name: "new-report", Title: "New Report", Evaluation: "1 == 1", Active: true,
		},
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if id != 100 {
		t.Errorf("Create() id = %d, want 100", id)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

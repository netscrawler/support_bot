package store

import (
	"context"
	"support_bot/internal/db/sqlcgen"
	"testing"

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

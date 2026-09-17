package store

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// NewReportStoreForTest builds a ReportStore around an already-constructed
// *sqlcgen.Queries (e.g. one backed by a pgxmock pool) for unit tests that
// don't need ExecTx/transaction machinery. Production code always uses
// NewReportStore.
func NewReportStoreForTest(q *sqlcgen.Queries) *ReportStore {
	return &ReportStore{q: q, log: slog.Default()}
}

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
		}).AddRow(
			"recipient-1",
			[]byte(`{}`),
			new("remote/path"),
			new(int32(555)),
			new(int32(7)),
			new("tg"),
			new(true),
			[]string{"a@example.com"},
			[]string{"b@example.com"},
			new("Subject"),
			new("Body text"),
			new(int64(99)),
			new("Chat Title"),
			new("tg"),
			new("Chat Description"),
			new(true),
			new("group"),
		))

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
		}).AddRow(
			new("csv"),
			new("report.csv"),
			new(int32(3)),
			new("Template One"),
			new("html"),
			new("<html></html>"),
			[]byte(`{"sheet1":["col1","col2"]}`),
		))

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

	if len(report.Recipients) != 1 {
		t.Fatalf("Recipients = %v, want 1 entry", report.Recipients)
	}
	recipient := report.Recipients[0]
	if recipient.Chat == nil || recipient.Chat.ChatID != 99 {
		t.Errorf("Recipients[0].Chat = %+v, want ChatID=99", recipient.Chat)
	}
	if recipient.Chat != nil &&
		(recipient.Chat.Title == nil || *recipient.Chat.Title != "Chat Title") {
		t.Errorf("Recipients[0].Chat.Title = %v, want %q", recipient.Chat.Title, "Chat Title")
	}
	if recipient.Email == nil || len(recipient.Email.Dest) != 1 ||
		recipient.Email.Dest[0] != "a@example.com" {
		t.Errorf("Recipients[0].Email = %+v, want Dest=[a@example.com]", recipient.Email)
	}
	if recipient.Type != models.RecipientType("tg") {
		t.Errorf("Recipients[0].Type = %q, want %q", recipient.Type, "tg")
	}
	if !recipient.NeedDeleteAfterEndOfDay {
		t.Errorf("Recipients[0].NeedDeleteAfterEndOfDay = false, want true")
	}
	if recipient.ThreadID == nil || *recipient.ThreadID != 555 {
		t.Errorf("Recipients[0].ThreadID = %v, want 555", recipient.ThreadID)
	}

	if len(report.Exports) != 1 {
		t.Fatalf("Exports = %v, want 1 entry", report.Exports)
	}
	export := report.Exports[0]
	if export.Format != "csv" {
		t.Errorf("Exports[0].Format = %q, want %q", export.Format, "csv")
	}
	if export.Template == nil || export.Template.ID != 3 ||
		export.Template.Title != "Template One" {
		t.Errorf("Exports[0].Template = %+v, want ID=3 Title=%q", export.Template, "Template One")
	}
	if len(export.Order["sheet1"]) != 2 || export.Order["sheet1"][0] != "col1" {
		t.Errorf("Exports[0].Order[sheet1] = %v, want [col1 col2]", export.Order["sheet1"])
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

// TestReportStore_LoadPaged_ClampsPage проверяет ограничение номера страницы
// в LoadPaged: страница <= 0 приводится к первой, а страница за пределами
// диапазона — к последней доступной, при этом возвращаются реальные строки
// этой страницы, а не пустой срез.
func TestReportStore_LoadPaged_ClampsPage(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		total        int64
		wantOffset   int32
		mockRowNames []string
	}{
		{
			// Вход: page = 0 при total = 12 (3 страницы по 5).
			// Ожидание: page приводится к 1, offset = 0.
			name:         "страница меньше или равна нулю приводится к первой странице",
			page:         0,
			total:        12,
			wantOffset:   0,
			mockRowNames: []string{"r1", "r2", "r3", "r4", "r5"},
		},
		{
			// Вход: page = 100 при total = 12 (последняя валидная страница — 3).
			// Ожидание: page приводится к 3, offset = 10, возвращаются
			// реальные строки последней страницы, а не пустой срез.
			name:         "страница за пределами диапазона возвращает последнюю страницу",
			page:         100,
			total:        12,
			wantOffset:   10,
			mockRowNames: []string{"r11", "r12"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock.NewPool() error = %v", err)
			}
			defer pool.Close()

			pool.ExpectQuery("select count\\(\\*\\) from reports where access_from_lk = true").
				WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(tt.total))

			rows := pgxmock.NewRows([]string{"id", "name", "title"})
			for i, name := range tt.mockRowNames {
				rows.AddRow(int32(i+1), name, "Title "+name)
			}

			pool.ExpectQuery("select id, name, title from reports where access_from_lk = true").
				WithArgs(int32(5), tt.wantOffset).
				WillReturnRows(rows)

			s := NewReportStoreForTest(sqlcgen.New(pool))

			reports, total, err := s.LoadPaged(context.Background(), tt.page)
			if err != nil {
				t.Fatalf("LoadPaged() error = %v", err)
			}

			if total != int(tt.total) {
				t.Errorf("LoadPaged() total = %d, want %d", total, tt.total)
			}
			if len(reports) != len(tt.mockRowNames) {
				t.Fatalf(
					"LoadPaged() returned %d reports, want %d (must not be empty for an out-of-range page)",
					len(reports),
					len(tt.mockRowNames),
				)
			}
			for i, name := range tt.mockRowNames {
				if reports[i].Name != name {
					t.Errorf("LoadPaged() reports[%d].Name = %q, want %q", i, reports[i].Name, name)
				}
			}

			if err := pool.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
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

// TestReportStore_Create_WiresAllDependencies проверяет всю ~250-строчную
// цепочку get-or-create в createWithPool для отчёта, у которого заполнены
// все виды зависимостей разом: карточка (query) с непустым card.Params,
// экспорт с шаблоном, крон и получатель с одновременно чатом и email.
//
// Вход: models.Report с одной карточкой (Params без RawParams), одним
// экспортом с шаблоном, одним кроном и одним получателем (чат + email);
// ни одна зависимость ещё не существует в БД (все Find* возвращают
// pgx.ErrNoRows), поэтому createWithPool должен создать (Create*) и
// привязать (Link*) каждую из них.
//
// Ожидание: полная и точная последовательность SQL-вызовов внутри
// транзакции, и, что важно для критического бага, аргумент params у
// CreateQuery равен json.Marshal(card.Params) — а не литералу "{}", в
// который параметры карточки молча превращались до фикса, потому что
// card.RawParams всегда nil на пути создания отчёта из JSON/DSL.
func TestReportStore_Create_WiresAllDependencies(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	card := models.Card{
		CardUUID: "card-uuid-1",
		Title:    "Card One",
		Type:     "mb",
		Params:   map[string]string{"date1": "2024-01-01", "date2": "2024-01-31"},
	}
	wantQueryParams, err := json.Marshal(card.Params)
	if err != nil {
		t.Fatalf("json.Marshal(card.Params) error = %v", err)
	}

	fileName := "report.csv"
	export := models.Export{
		Format:   "csv",
		FileName: &fileName,
		Order:    map[string][]string{"sheet1": {"col1", "col2"}},
		Template: &models.Template{Title: "Tmpl One", Type: "html", TemplateText: "<html></html>"},
	}
	wantSortOrder, err := json.Marshal(export.Order)
	if err != nil {
		t.Fatalf("json.Marshal(export.Order) error = %v", err)
	}

	cron := models.Cron{
		Name:        "daily",
		Cron:        "0 0 * * *",
		Description: "desc",
		IsActive:    true,
		EventType:   1,
	}

	chatTitle, chatDesc := "Chat Title", "Chat Desc"
	chat := models.Chat{
		ChatID:      555,
		Title:       &chatTitle,
		Type:        "tg",
		Description: &chatDesc,
		IsActive:    true,
		ChType:      "tg",
	}

	emailBody := "Body text"
	email := models.EmailTemplate{
		Dest: []string{"a@example.com"}, Copy: []string{"b@example.com"},
		Subject: "Subject", Body: &emailBody,
	}

	threadID := 7
	recipient := models.Recipient{
		Name: "recipient-1", Chat: &chat, ThreadID: &threadID, Email: &email,
		Type: models.RecipientType("tg"), NeedDeleteAfterEndOfDay: true,
	}

	rep := models.Report{
		Name: "full-report", Title: "Full Report", Evaluation: "1 == 1", Active: true,
		Queries:    []models.Card{card},
		Exports:    []models.Export{export},
		Crons:      []models.Cron{cron},
		Recipients: []models.Recipient{recipient},
	}

	pool.ExpectBegin()

	pool.ExpectQuery("select exists").
		WithArgs("full-report").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	pool.ExpectQuery("select id from evaluate").
		WithArgs("1 == 1").
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("insert into evaluate").
		WithArgs("1 == 1").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(10)))

	pool.ExpectQuery("insert into reports").
		WithArgs("full-report", "Full Report", int64(10), (*int64)(nil), false, true).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(100)))

	// Карточка: не найдена -> создаётся с промаршаленными card.Params.
	pool.ExpectQuery("select id from queries").
		WithArgs("card-uuid-1", "Card One").
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("insert into queries").
		WithArgs("card-uuid-1", "Card One", "mb", wantQueryParams).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(21)))
	pool.ExpectExec("insert into report_queries").
		WithArgs(int32(100), int32(21)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Экспорт + формат + шаблон.
	pool.ExpectQuery("select id from export_formats").
		WithArgs(new("csv")).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("insert into export_formats").
		WithArgs(new("csv")).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(31)))
	pool.ExpectExec("insert into reports_export").
		WithArgs(new(int32(100)), new(int32(31)), &fileName, wantSortOrder).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	pool.ExpectQuery("select id from templates").
		WithArgs(new("Tmpl One"), "html").
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("insert into templates").
		WithArgs(new("<html></html>"), new("Tmpl One"), "html").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(41)))
	pool.ExpectExec("insert into report_templates").
		WithArgs(int32(100), int32(41)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Крон.
	pool.ExpectQuery("select id from crons").
		WithArgs("daily", "0 0 * * *").
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("insert into crons").
		WithArgs("0 0 * * *", "daily", new("desc"), true, int32(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(51)))
	pool.ExpectExec("insert into report_crons").
		WithArgs(int32(100), int32(51)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Получатель: сначала чат, потом email, потом сам получатель.
	pool.ExpectQuery("select id from recipients").
		WithArgs("recipient-1").
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("select id from chats").
		WithArgs(int64(555)).
		WillReturnError(pgx.ErrNoRows)
	pool.ExpectQuery("insert into chats").
		WithArgs(int64(555), &chatTitle, "tg", &chatDesc, true, "tg").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(61)))
	pool.ExpectQuery("insert into email_templates").
		WithArgs([]string{"a@example.com"}, []string{"b@example.com"}, "Subject", &emailBody).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(71)))
	pool.ExpectQuery("insert into recipients").
		WithArgs("recipient-1", (*string)(nil), new(int32(61)), new(int32(7)), new(int32(71)), new("tg"), new(true)).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int32(81)))
	pool.ExpectExec("insert into reports_recipients").
		WithArgs(new(int32(100)), new(int32(81))).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	pool.ExpectCommit()

	id, err := execTxOnMock(t, pool, NewReportStore(nil, slog.New(slog.DiscardHandler)), rep)
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

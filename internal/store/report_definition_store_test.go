package store

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

// NewReportDefinitionStoreForTest строит ReportDefinitionStore поверх уже
// созданного *sqlcgen.Queries (например, на основе pgxmock-пула) для
// модульных тестов. Продакшен-код всегда использует NewReportDefinitionStore.
func NewReportDefinitionStoreForTest(q *sqlcgen.Queries) *ReportDefinitionStore {
	return &ReportDefinitionStore{q: q, log: slog.Default()}
}

func sampleLayout() *models.Layout {
	return &models.Layout{
		Version: 1,
		Page: models.PageConfig{
			Format:      "A4",
			Orientation: "portrait",
			PaddingMM:   16,
		},
		Blocks: []models.Block{
			{
				ID:      "daily-summary",
				Type:    models.BlockTypeMetric,
				Dataset: "daily",
				Pos:     models.GridPosition{X: 0, Y: 0, W: 3, H: 2},
			},
		},
	}
}

// TestReportDefinitionStore_Create_Success проверяет, что Create передаёт
// все 11 полей модели в CreateReportDefinition в правильном порядке,
// включая сериализованный layout и валидные timestamptz-значения.
func TestReportDefinitionStore_Create_Success(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	layout := sampleLayout()
	layoutBytes, err := json.Marshal(layout)
	if err != nil {
		t.Fatalf("json.Marshal(layout) error = %v", err)
	}

	createdAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)

	def := &models.ReportDefinition{
		ID:          "def-1",
		Name:        "Daily report",
		Description: "generated daily",
		Layout:      layout,
		Format:      "pdf",
		Status:      models.ReportStatusPending,
		FilePath:    "/tmp/def-1.pdf",
		ErrorMsg:    "",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		ExpiresAt:   &expiresAt,
	}

	pool.ExpectExec("insert into report_definitions").
		WithArgs(
			"def-1",
			"Daily report",
			"generated daily",
			layoutBytes,
			"pdf",
			string(models.ReportStatusPending),
			"/tmp/def-1.pdf",
			"",
			pgtype.Timestamptz{Time: createdAt, Valid: true},
			pgtype.Timestamptz{Time: updatedAt, Valid: true},
			pgtype.Timestamptz{Time: expiresAt, Valid: true},
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	if err := s.Create(context.Background(), def); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_Create_NilLayoutSendsSQLNull проверяет, что
// nil-Layout передаётся в insert как настоящий SQL NULL (nil []byte), а не
// как JSON-байты "null" — json.Marshal((*models.Layout)(nil)) даёт "null",
// что при чтении обратно превратилось бы в не-nil *Layout.
func TestReportDefinitionStore_Create_NilLayoutSendsSQLNull(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	createdAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	def := &models.ReportDefinition{
		ID:          "def-2",
		Name:        "No layout report",
		Description: "",
		Layout:      nil,
		Format:      "csv",
		Status:      models.ReportStatusPending,
		FilePath:    "",
		ErrorMsg:    "",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		ExpiresAt:   nil,
	}

	pool.ExpectExec("insert into report_definitions").
		WithArgs(
			"def-2",
			"No layout report",
			"",
			[]byte(nil),
			"csv",
			string(models.ReportStatusPending),
			"",
			"",
			pgtype.Timestamptz{Time: createdAt, Valid: true},
			pgtype.Timestamptz{Time: updatedAt, Valid: true},
			pgtype.Timestamptz{},
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	if err := s.Create(context.Background(), def); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_GetByID_ReturnsPopulatedDefinition проверяет,
// что GetByID разворачивает JSON-колонку layout обратно в *models.Layout
// поле за полем.
func TestReportDefinitionStore_GetByID_ReturnsPopulatedDefinition(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	layout := sampleLayout()
	layoutBytes, err := json.Marshal(layout)
	if err != nil {
		t.Fatalf("json.Marshal(layout) error = %v", err)
	}

	createdAt := pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true}
	updatedAt := pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Valid: true}

	pool.ExpectQuery("select id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at").
		WithArgs("def-1").
		WillReturnRows(pgxmock.NewRows(
			[]string{
				"id", "name", "description", "layout", "format", "status",
				"file_path", "error_msg", "created_at", "updated_at", "expires_at",
			},
		).AddRow(
			"def-1", "Daily report", "generated daily", layoutBytes, "pdf",
			"pending", "/tmp/def-1.pdf", "", createdAt, updatedAt, pgtype.Timestamptz{},
		))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	def, err := s.GetByID(context.Background(), "def-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if def == nil {
		t.Fatal("GetByID() returned nil definition, want a populated *models.ReportDefinition")
	}
	if def.Layout == nil {
		t.Fatal("GetByID() Layout = nil, want populated layout")
	}
	if def.Layout.Version != layout.Version || def.Layout.Page.Format != layout.Page.Format {
		t.Errorf("GetByID() Layout = %+v, want %+v", def.Layout, layout)
	}
	if len(def.Layout.Blocks) != 1 || def.Layout.Blocks[0].ID != "daily-summary" {
		t.Errorf("GetByID() Layout.Blocks = %+v, want a single block with ID=daily-summary", def.Layout.Blocks)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_GetByID_NotFound_ReturnsErrNotFound проверяет,
// что pgx.ErrNoRows транслируется в models.ErrNotFound, распознаваемый через
// errors.Is даже после оборачивания через fmt.Errorf.
func TestReportDefinitionStore_GetByID_NotFound_ReturnsErrNotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("select id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at").
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	def, err := s.GetByID(context.Background(), "missing")
	if def != nil {
		t.Errorf("GetByID() = %+v, want nil", def)
	}
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want wrapping models.ErrNotFound", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_GetByID_NilLayoutColumnStaysNil проверяет, что
// пустая колонка layout (NULL) даёт def.Layout == nil, а не &models.Layout{}.
func TestReportDefinitionStore_GetByID_NilLayoutColumnStaysNil(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	createdAt := pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true}
	updatedAt := pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Valid: true}

	pool.ExpectQuery("select id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at").
		WithArgs("def-3").
		WillReturnRows(pgxmock.NewRows(
			[]string{
				"id", "name", "description", "layout", "format", "status",
				"file_path", "error_msg", "created_at", "updated_at", "expires_at",
			},
		).AddRow(
			"def-3", "No layout", "", []byte(nil), "csv",
			"pending", "", "", createdAt, updatedAt, pgtype.Timestamptz{},
		))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	def, err := s.GetByID(context.Background(), "def-3")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if def.Layout != nil {
		t.Errorf("GetByID() Layout = %+v, want nil", def.Layout)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_Update_Success проверяет порядок и значения всех
// аргументов UpdateReportDefinition — по полю UpdateReportDefinitionParams
// сгенерированного sqlc-кода: id, name, description, layout, format, status,
// file_path, error_msg, updated_at, expires_at.
func TestReportDefinitionStore_Update_Success(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	layout := sampleLayout()
	layoutBytes, err := json.Marshal(layout)
	if err != nil {
		t.Fatalf("json.Marshal(layout) error = %v", err)
	}

	updatedAt := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2024, 2, 8, 0, 0, 0, 0, time.UTC)

	def := &models.ReportDefinition{
		ID:          "def-1",
		Name:        "Daily report v2",
		Description: "updated",
		Layout:      layout,
		Format:      "pdf",
		Status:      models.ReportStatusCompleted,
		FilePath:    "/tmp/def-1-v2.pdf",
		ErrorMsg:    "",
		UpdatedAt:   updatedAt,
		ExpiresAt:   &expiresAt,
	}

	pool.ExpectExec("update report_definitions").
		WithArgs(
			"def-1",
			"Daily report v2",
			"updated",
			layoutBytes,
			"pdf",
			string(models.ReportStatusCompleted),
			"/tmp/def-1-v2.pdf",
			"",
			pgtype.Timestamptz{Time: updatedAt, Valid: true},
			pgtype.Timestamptz{Time: expiresAt, Valid: true},
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	if err := s.Update(context.Background(), def); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_ListByStatus_ReturnsRows проверяет привязку
// аргументов status/limit и корректное отображение всех строк результата.
func TestReportDefinitionStore_ListByStatus_ReturnsRows(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	createdAt := pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true}
	updatedAt := pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Valid: true}

	pool.ExpectQuery("select id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at").
		WithArgs("pending", int32(10)).
		WillReturnRows(pgxmock.NewRows(
			[]string{
				"id", "name", "description", "layout", "format", "status",
				"file_path", "error_msg", "created_at", "updated_at", "expires_at",
			},
		).
			AddRow("def-1", "Report One", "", []byte(nil), "pdf", "pending", "", "", createdAt, updatedAt, pgtype.Timestamptz{}).
			AddRow("def-2", "Report Two", "", []byte(nil), "csv", "pending", "", "", createdAt, updatedAt, pgtype.Timestamptz{}),
		)

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	defs, err := s.ListByStatus(context.Background(), models.ReportStatusPending, 10)
	if err != nil {
		t.Fatalf("ListByStatus() error = %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("ListByStatus() returned %d definitions, want 2", len(defs))
	}
	if defs[0].ID != "def-1" || defs[1].ID != "def-2" {
		t.Errorf("ListByStatus() = %+v, want IDs def-1, def-2 in order", defs)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_Delete_Success проверяет единственный exec с
// аргументом id.
func TestReportDefinitionStore_Delete_Success(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectExec("delete from report_definitions where id = \\$1").
		WithArgs("def-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	if err := s.Delete(context.Background(), "def-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_CleanupExpired_ReturnsRowCount проверяет, что
// возвращается реальное количество удалённых строк.
func TestReportDefinitionStore_CleanupExpired_ReturnsRowCount(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectExec("delete from report_definitions").
		WillReturnResult(pgxmock.NewResult("DELETE", 3))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	n, err := s.CleanupExpired(context.Background())
	if err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if n != 3 {
		t.Errorf("CleanupExpired() = %d, want 3", n)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestReportDefinitionStore_ExpiresAt_NilRoundTrips проверяет оба направления
// для nil ExpiresAt: Create должен передать pgtype.Timestamptz{Valid: false},
// а GetByID с Valid: false в колонке должен дать def.ExpiresAt == nil.
func TestReportDefinitionStore_ExpiresAt_NilRoundTrips(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	createdAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	def := &models.ReportDefinition{
		ID:          "def-4",
		Name:        "No expiry",
		Description: "",
		Layout:      nil,
		Format:      "csv",
		Status:      models.ReportStatusPending,
		FilePath:    "",
		ErrorMsg:    "",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		ExpiresAt:   nil,
	}

	pool.ExpectExec("insert into report_definitions").
		WithArgs(
			"def-4",
			"No expiry",
			"",
			[]byte(nil),
			"csv",
			string(models.ReportStatusPending),
			"",
			"",
			pgtype.Timestamptz{Time: createdAt, Valid: true},
			pgtype.Timestamptz{Time: updatedAt, Valid: true},
			pgtype.Timestamptz{Valid: false},
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewReportDefinitionStoreForTest(sqlcgen.New(pool))

	if err := s.Create(context.Background(), def); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	pool.ExpectQuery("select id, name, description, layout, format, status, file_path, error_msg, created_at, updated_at, expires_at").
		WithArgs("def-4").
		WillReturnRows(pgxmock.NewRows(
			[]string{
				"id", "name", "description", "layout", "format", "status",
				"file_path", "error_msg", "created_at", "updated_at", "expires_at",
			},
		).AddRow(
			"def-4", "No expiry", "", []byte(nil), "csv", "pending", "", "",
			pgtype.Timestamptz{Time: createdAt, Valid: true},
			pgtype.Timestamptz{Time: updatedAt, Valid: true},
			pgtype.Timestamptz{Valid: false},
		))

	got, err := s.GetByID(context.Background(), "def-4")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ExpiresAt != nil {
		t.Errorf("GetByID() ExpiresAt = %v, want nil", got.ExpiresAt)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

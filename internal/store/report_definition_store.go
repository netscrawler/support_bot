package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportDefinitionStore хранит записи рендеринга экспорт-пайплайна
// (models.ReportDefinition, таблица report_definitions) — это отдельный,
// не связанный с ReportStore (report_store.go) домен: ReportStore обслуживает
// DSL-конфигурацию отчётов (models.Report, таблица reports), а
// ReportDefinitionStore — сохранённые результаты рендера конкретного
// экспорта (layout, статус, путь к файлу, срок жизни).
type ReportDefinitionStore struct {
	q   *sqlcgen.Queries
	log *slog.Logger
}

func NewReportDefinitionStore(pool *pgxpool.Pool, log *slog.Logger) *ReportDefinitionStore {
	return &ReportDefinitionStore{
		q:   sqlcgen.New(pool),
		log: log.With(slog.Any("module", "store.report_definition")),
	}
}

func (s *ReportDefinitionStore) Create(ctx context.Context, def *models.ReportDefinition) error {
	layout, err := marshalLayout(def.Layout)
	if err != nil {
		return fmt.Errorf("create report definition: %w", err)
	}

	err = s.q.CreateReportDefinition(ctx, sqlcgen.CreateReportDefinitionParams{
		ID:          def.ID,
		Name:        def.Name,
		Description: def.Description,
		Layout:      layout,
		Format:      def.Format,
		Status:      string(def.Status),
		FilePath:    def.FilePath,
		ErrorMsg:    def.ErrorMsg,
		CreatedAt:   toTimestamptz(def.CreatedAt),
		UpdatedAt:   toTimestamptz(def.UpdatedAt),
		ExpiresAt:   toTimestamptzOrNull(def.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("create report definition: %w", err)
	}

	return nil
}

func (s *ReportDefinitionStore) GetByID(ctx context.Context, id string) (*models.ReportDefinition, error) {
	row, err := s.q.GetReportDefinitionByID(ctx, id)
	if err != nil {
		return nil, translateNoRows(fmt.Errorf("get report definition %s: %w", id, err))
	}

	def, err := assemble(row)
	if err != nil {
		return nil, err
	}

	return def, nil
}

func (s *ReportDefinitionStore) Update(ctx context.Context, def *models.ReportDefinition) error {
	layout, err := marshalLayout(def.Layout)
	if err != nil {
		return fmt.Errorf("update report definition: %w", err)
	}

	err = s.q.UpdateReportDefinition(ctx, sqlcgen.UpdateReportDefinitionParams{
		ID:          def.ID,
		Name:        def.Name,
		Description: def.Description,
		Layout:      layout,
		Format:      def.Format,
		Status:      string(def.Status),
		FilePath:    def.FilePath,
		ErrorMsg:    def.ErrorMsg,
		UpdatedAt:   toTimestamptz(def.UpdatedAt),
		ExpiresAt:   toTimestamptzOrNull(def.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("update report definition: %w", err)
	}

	return nil
}

func (s *ReportDefinitionStore) ListByStatus(
	ctx context.Context,
	status models.ReportStatus,
	limit int,
) ([]models.ReportDefinition, error) {
	rows, err := s.q.ListReportDefinitionsByStatus(ctx, sqlcgen.ListReportDefinitionsByStatusParams{
		Status: string(status),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list report definitions by status %s: %w", status, err)
	}

	defs := make([]models.ReportDefinition, 0, len(rows))

	for _, row := range rows {
		def, err := assemble(row)
		if err != nil {
			return nil, err
		}

		defs = append(defs, *def)
	}

	return defs, nil
}

func (s *ReportDefinitionStore) Delete(ctx context.Context, id string) error {
	if err := s.q.DeleteReportDefinition(ctx, id); err != nil {
		return fmt.Errorf("delete report definition: %w", err)
	}

	return nil
}

func (s *ReportDefinitionStore) CleanupExpired(ctx context.Context) (int64, error) {
	n, err := s.q.CleanupExpiredReportDefinitions(ctx)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired report definitions: %w", err)
	}

	return n, nil
}

func assemble(row sqlcgen.ReportDefinition) (*models.ReportDefinition, error) {
	layout, err := unmarshalLayout(row.Layout)
	if err != nil {
		return nil, fmt.Errorf("assemble report definition %s: %w", row.ID, err)
	}

	return &models.ReportDefinition{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Layout:      layout,
		Format:      row.Format,
		Status:      models.ReportStatus(row.Status),
		FilePath:    row.FilePath,
		ErrorMsg:    row.ErrorMsg,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		ExpiresAt:   fromTimestamptzPtr(row.ExpiresAt),
	}, nil
}

// marshalLayout сериализует layout в JSON для колонки layout. nil-указатель
// осознанно возвращает nil []byte (а не json-байты "null"): pgx интерпретирует
// nil []byte как настоящий SQL NULL, тогда как json.Marshal((*Layout)(nil))
// дал бы 4 байта "null", которые при чтении обратно превратились бы в
// не-nil *Layout с нулевыми полями — то есть неотличимо от "layout когда-то
// был задан, но пустой".
func marshalLayout(l *models.Layout) ([]byte, error) {
	if l == nil {
		return nil, nil
	}

	return json.Marshal(l)
}

func unmarshalLayout(b []byte) (*models.Layout, error) {
	if len(b) == 0 {
		return nil, nil
	}

	var l models.Layout
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("unmarshal layout: %w", err)
	}

	return &l, nil
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toTimestamptzOrNull(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromTimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}

	return &t.Time
}

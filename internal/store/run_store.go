package store

import (
	"context"
	"errors"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// maxRunErrorRunes — предел длины текста ошибки в журнале запусков.
const maxRunErrorRunes = 4000

// RunStore — журнал запусков отчётов поверх sqlc-запросов report_run.sql.
type RunStore struct {
	q *sqlcgen.Queries
}

func NewRunStore(pool *pgxpool.Pool) *RunStore {
	return &RunStore{q: sqlcgen.New(pool)}
}

// Insert пишет запуск; текст ошибки режется по рунам, пустые строки -> NULL.
func (s *RunStore) Insert(ctx context.Context, r models.Run) error {
	if rs := []rune(r.Error); len(rs) > maxRunErrorRunes {
		r.Error = string(rs[:maxRunErrorRunes])
	}

	return s.q.InsertReportRun(ctx, sqlcgen.InsertReportRunParams{
		ReportName: r.ReportName,
		Kind:       string(r.Kind),
		Status:     string(r.Status),
		Reason:     nilIfEmpty(r.Reason),
		Error:      nilIfEmpty(r.Error),
		ChatID:     r.ChatID,
		StartedAt:  pgtype.Timestamptz{Time: r.StartedAt, Valid: true},
		FinishedAt: pgtype.Timestamptz{Time: r.FinishedAt, Valid: true},
	})
}

// LastStatus — статус последнего cron/manual запуска; ok=false, если запусков нет.
func (s *RunStore) LastStatus(
	ctx context.Context,
	reportName string,
) (models.RunStatus, bool, error) {
	st, err := s.q.LastReportRunStatus(ctx, reportName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return models.RunStatus(st), true, nil
}

func (s *RunStore) List(ctx context.Context, reportName string, limit int) ([]models.Run, error) {
	rows, err := s.q.ListReportRuns(ctx, sqlcgen.ListReportRunsParams{
		ReportName: reportName,
		Limit:      int32(limit), //nolint:gosec // limit мал
	})
	if err != nil {
		return nil, err
	}

	runs := make([]models.Run, 0, len(rows))
	for _, r := range rows {
		runs = append(runs, models.Run{
			ReportName: r.ReportName,
			Kind:       models.RunKind(r.Kind),
			Status:     models.RunStatus(r.Status),
			Reason:     derefStr(r.Reason),
			Error:      derefStr(r.Error),
			ChatID:     r.ChatID,
			StartedAt:  r.StartedAt.Time,
			FinishedAt: r.FinishedAt.Time,
		})
	}

	return runs, nil
}

func (s *RunStore) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return s.q.DeleteReportRunsBefore(ctx, pgtype.Timestamptz{Time: before, Valid: true})
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

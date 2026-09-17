package store

import (
	"context"
	"fmt"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CronStore хранит расписания (таблица crons) и их связи с отчётами
// (таблица report_crons): активные расписания читает планировщик
// (internal/sheduler), а привязки cron→отчёт — event_creator.
type CronStore struct{ q *sqlcgen.Queries }

func NewCronStore(pool *pgxpool.Pool) *CronStore {
	return &CronStore{q: sqlcgen.New(pool)}
}

// LoadActive возвращает только расписания с is_active = true — неактивные
// crons планировщику не нужны.
func (s *CronStore) LoadActive(ctx context.Context) ([]models.SheduleUnit, error) {
	rows, err := s.q.ListActiveCrons(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active crons: %w", err)
	}

	units := make([]models.SheduleUnit, 0, len(rows))
	for _, row := range rows {
		units = append(units, mapCronRow(row))
	}

	return units, nil
}

// LoadEvents возвращает привязки cron→отчёт только для отчётов с
// active = true — событие для неактивного отчёта создавать не нужно.
func (s *CronStore) LoadEvents(ctx context.Context) ([]models.ReportCronEvent, error) {
	rows, err := s.q.ListEventsForActiveReports(ctx)
	if err != nil {
		return nil, fmt.Errorf("load events for active reports: %w", err)
	}

	return mapReportCronEvents(rows), nil
}

// LoadEventsByCronName возвращает привязки cron→отчёт для одного расписания
// по имени, также только для отчётов с active = true.
func (s *CronStore) LoadEventsByCronName(
	ctx context.Context,
	name string,
) ([]models.ReportCronEvent, error) {
	rows, err := s.q.ListEventsForActiveReportsByCronName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("load events for active reports by cron name: %w", err)
	}

	events := make([]models.ReportCronEvent, 0, len(rows))
	for _, row := range rows {
		events = append(
			events,
			models.ReportCronEvent{CronName: row.CronName, Name: row.ReportName},
		)
	}

	return events, nil
}

func mapCronRow(row sqlcgen.ListActiveCronsRow) models.SheduleUnit {
	return models.SheduleUnit{Crontab: row.Cron, Name: row.Name, EventType: int(row.EventType)}
}

func mapReportCronEvents(rows []sqlcgen.ListEventsForActiveReportsRow) []models.ReportCronEvent {
	events := make([]models.ReportCronEvent, 0, len(rows))
	for _, row := range rows {
		events = append(
			events,
			models.ReportCronEvent{CronName: row.CronName, Name: row.ReportName},
		)
	}

	return events
}

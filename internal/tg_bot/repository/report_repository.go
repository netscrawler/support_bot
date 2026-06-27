package repository

import (
	"context"
	"log/slog"
	models "support_bot/internal/models/report"

	"github.com/jmoiron/sqlx"
)

type ReportRepository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewReportRepository(db *sqlx.DB, log *slog.Logger) *ReportRepository {
	l := log.With(slog.Any("module", "tg_bot.repository.report"))

	return &ReportRepository{db: db, log: l}
}

type report struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Title string `db:"title"`
}

func (r *ReportRepository) LoadReports(ctx context.Context, page int) ([]models.ReportForTgLK, error) {
	const query = `select id, name, title from reports where access_from_lk = true order by id limit $1 offset $2`
	const limit = 5

	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	var reports []report

	err := r.db.SelectContext(ctx, &reports, query, limit, offset)
	if err != nil {
		return nil, err
	}

	reportLK := make([]models.ReportForTgLK, 0, len(reports))

	for _, r := range reports {
		reportLK = append(reportLK, models.ReportForTgLK{
			ID:    r.ID,
			Name:  r.Name,
			Title: r.Title,
		})
	}

	return reportLK, nil
}

func (r *ReportRepository) GetReportsCount(ctx context.Context) (int, error) {
	const query = `select count(*) from reports where access_from_lk = true`
	var count int
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

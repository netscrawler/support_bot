package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"support_bot/internal/models"
)

type ReportDB interface {
	GetByPublicID(ctx context.Context, reportID string) (*models.Report, error)
}

type ReportGenerator interface {
	Generate(ctx context.Context, report models.Report) (models.Data, error)
}

type Report struct {
	db  ReportDB
	gen ReportGenerator
	log *slog.Logger
}

func NewReport(db ReportDB, gen ReportGenerator, log *slog.Logger) *Report {
	return &Report{db: db, gen: gen, log: log.With(slog.Any("module", "report_generator"))}
}

func (r *Report) GenerateReport(ctx context.Context, reportID string) (models.Data, error) {
	report, err := r.db.GetByPublicID(ctx, reportID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.Data{}, models.ErrNotFound
		}

		return models.Data{}, fmt.Errorf("get report: %w", err)
	}

	data, err := r.gen.Generate(ctx, *report)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.Data{}, models.ErrNotFound
		}

		return models.Data{},
			fmt.Errorf("%w: generate report: %w", models.ErrInternal, err)
	}

	return data, nil
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"support_bot/internal/models"
)

type ReportStore interface {
	Load(ctx context.Context) ([]models.Report, error)
	Create(ctx context.Context, report models.Report) (int64, error)
}

type ReportManager struct {
	store ReportStore
	val   *ReportValidation
	log   *slog.Logger
}

func NewReportManager(store ReportStore, val *ReportValidation, log *slog.Logger) *ReportManager {
	return &ReportManager{
		store: store,
		val:   val,
		log:   log.With("module", "report_manager"),
	}
}

func (m *ReportManager) Load(ctx context.Context) ([]models.Report, error) {
	m.log.InfoContext(ctx, "Loading all reports")

	return m.store.Load(ctx)
}

func (m *ReportManager) Create(ctx context.Context, repr io.Reader) error {
	var report models.Report
	if err := json.NewDecoder(repr).Decode(&report); err != nil {
		m.log.ErrorContext(ctx, "Failed to unmarshal JSON", slog.Any("error", err))

		return err
	}

	m.log.InfoContext(ctx, "Creating report", slog.String("name", report.Name))

	if err := m.val.Validate(ctx, report); err != nil {
		return fmt.Errorf("validate report: %w", err)
	}

	if _, err := m.store.Create(ctx, report); err != nil {
		if errors.Is(err, models.ErrAlreadyExist) {
			return fmt.Errorf("report %q already exists: %w", report.Name, err)
		}

		return fmt.Errorf("create report: %w", err)
	}

	m.log.InfoContext(ctx, "Report successfully created", slog.String("name", report.Name))

	return nil
}

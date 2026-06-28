package service

import (
	"context"
	"fmt"
	"log/slog"

	eventcreator "support_bot/internal/event_creator"
	models2 "support_bot/internal/models"
	"support_bot/internal/sheduler"
	"support_bot/internal/tg_bot/repository"
)

type Report struct {
	*sheduler.SheduleAPI
	*eventcreator.EventAPI

	repo *repository.ReportRepository

	log *slog.Logger
}

const reportsPageSize = 5

func NewReportService(
	shd *sheduler.SheduleAPI,
	eventAPI *eventcreator.EventAPI,
	repo *repository.ReportRepository,
	log *slog.Logger,
) *Report {
	l := log.With(slog.Any("module", "tg_bot.service.report"))

	return &Report{
		SheduleAPI: shd,
		EventAPI:   eventAPI,
		repo:       repo,
		log:        l,
	}
}

func (r *Report) LoadReportsWithPagination(ctx context.Context) (models2.LoadReportRPL, error) {
	return r.LoadReportByPage(ctx, 1)
}

func (r *Report) LoadReportByPage(ctx context.Context, page int) (models2.LoadReportRPL, error) {
	rCount, err := r.repo.GetReportsCount(ctx)
	if err != nil {
		return models2.LoadReportRPL{}, err
	}

	if rCount <= 0 {
		return models2.LoadReportRPL{}, fmt.Errorf("reports not found")
	}

	pageCount := (rCount + reportsPageSize - 1) / reportsPageSize

	if page <= 0 {
		page = 1
	}

	if page > pageCount {
		page = pageCount
	}

	reports, err := r.repo.LoadReports(ctx, page)
	if err != nil {
		return models2.LoadReportRPL{}, err
	}

	rpl := models2.LoadReportRPL{
		ReportsTotal: rCount,
		PageCount:    pageCount,
		CurrentPage:  page,
		Reports:      reports,
	}

	return rpl, nil
}

func (r *Report) GenerateReportByName(
	ctx context.Context,
	reportName string,
	chat *models2.Chat,
) error {
	rcpt := models2.Recipient{
		Name:                    "SpetialTGRcpt",
		Chat:                    chat,
		Type:                    models2.TelegramRecipient,
		NeedDeleteAfterEndOfDay: false,
	}
	r.ProduceSpecialEvent(ctx, reportName, rcpt)

	return nil
}

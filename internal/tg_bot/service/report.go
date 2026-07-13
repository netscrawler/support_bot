package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"support_bot/internal/event"
	"support_bot/internal/models"
	"support_bot/internal/sheduler"
	"support_bot/internal/tg_bot/repository"
)

type Report struct {
	*sheduler.SheduleAPI
	*event.EventAPI

	repo *repository.ReportRepository

	mbURL string

	log *slog.Logger
}

const reportsPageSize = 5

func NewReportService(
	shd *sheduler.SheduleAPI,
	eventAPI *event.EventAPI,
	repo *repository.ReportRepository,
	mbURL string,
	log *slog.Logger,
) *Report {
	l := log.With(slog.Any("module", "tg_bot.service.report"))

	return &Report{
		SheduleAPI: shd,
		EventAPI:   eventAPI,
		repo:       repo,
		mbURL:      mbURL,
		log:        l,
	}
}

func (r *Report) LoadReportsWithPagination(ctx context.Context) (models.LoadReportRPL, error) {
	return r.LoadReportByPage(ctx, 1)
}

func (r *Report) LoadReportByPage(ctx context.Context, page int) (models.LoadReportRPL, error) {
	rCount, err := r.repo.GetReportsCount(ctx)
	if err != nil {
		return models.LoadReportRPL{}, err
	}

	if rCount <= 0 {
		return models.LoadReportRPL{}, fmt.Errorf("reports not found")
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
		return models.LoadReportRPL{}, err
	}

	rpl := models.LoadReportRPL{
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
	chat *models.Chat,
) error {
	rcpt := models.Recipient{
		Name:                    "SpetialTGRcpt",
		Chat:                    chat,
		Type:                    models.TelegramRecipient,
		NeedDeleteAfterEndOfDay: false,
	}
	r.ProduceSpecialEvent(ctx, reportName, rcpt)

	return nil
}

func (r *Report) GenerateAndSendReportByName(
	ctx context.Context,
	reportName string,
) {
	r.ProduceGenEvent(ctx, reportName)
}

func (r *Report) GetReportInfoByName(
	ctx context.Context,
	reportName string,
) (models.ReportInfo, error) {
	rpt, err := r.repo.GetReportByName(ctx, reportName)
	if err != nil {
		return models.ReportInfo{}, fmt.Errorf("%w: (%w)", models.ErrInternal, err)
	}

	var cronString, recipientString, exportString, queryString []string

	crons, err := r.repo.GetReportLinkedCrons(ctx, reportName)
	if err != nil {
		cronString = []string{fmt.Sprintf("%s: (%s)", models.ErrInternal.Error(), err.Error())}
	}

	for _, cron := range crons {
		cronString = append(cronString, cron.String())
	}

	for _, rcpt := range rpt.Recipients {
		recipientString = append(recipientString, rcpt.String())
	}

	for _, exp := range rpt.Exports {
		exportString = append(exportString, exp.String())
	}

	for _, q := range rpt.Queries {
		queryString = append(
			queryString,
			fmt.Sprintf("<a href=\"%s\">%s</a>", q.GetFullURL(r.mbURL), q.Title),
		)
	}

	nextCronStr := ""

	if len(cronString) > 0 {
		nextCron, err := calculateNearestCronActivation(crons)
		if err != nil {
			nextCronStr = err.Error()
		} else {
			nextCronStr = nextCron.String()
		}
	}

	rpInf := models.ReportInfo{
		Name:       rpt.Name,
		Title:      rpt.Title,
		Evaluation: rpt.Evaluation,
		Recipients: recipientString,
		Exports:    exportString,
		Queries:    queryString,
		LinkedCron: cronString,
		NextCron:   nextCronStr,
	}

	return rpInf, nil
}

func calculateNearestCronActivation(units []models.SheduleUnit) (time.Time, error) {
	var nearest time.Time

	for _, u := range units {
		t, err := calculateNextCronActivation(u.Crontab)
		if err != nil {
			continue
		}

		if nearest.IsZero() || t.Before(nearest) {
			nearest = t
		}
	}

	if nearest.IsZero() {
		return time.Time{}, fmt.Errorf("error calculating next cron activation")
	}

	return nearest, nil
}

func calculateNextCronActivation(cronTab string) (time.Time, error) {
	cr, err := cron.ParseStandard(cronTab)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: (%w)", models.ErrInternal, err)
	}

	return cr.Next(time.Now()), nil
}

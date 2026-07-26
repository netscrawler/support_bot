package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"support_bot/internal/models"
)

type ReportRepository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewReportRepository(db *sqlx.DB, log *slog.Logger) *ReportRepository {
	l := log.With(slog.Any("module", "tg_bot.repository.report"))

	return &ReportRepository{db: db, log: l}
}

func (r *ReportRepository) LoadReports(
	ctx context.Context,
	page int,
) ([]models.ReportForTgLK, error) {
	const (
		query = `select id, name, title from reports where access_from_lk = true order by id limit $1 offset $2`
		limit = 5
	)

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

func (r *ReportRepository) GetReportByName(
	ctx context.Context,
	reportName string,
) (*models.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("report query: %w", ctx.Err())
	}

	rptByName, err := r.loadReportByName(ctx, reportName)
	if err != nil {
		r.log.ErrorContext(ctx, "error loading reports", slog.Any("error", err))

		return nil, err
	}

	rpt, err := r.getReportByID(ctx, *rptByName)
	if err != nil {
		return nil, fmt.Errorf("get report by id: %w", err)
	}

	return rpt, nil
}

func (r *ReportRepository) GetReportLinkedCrons(
	ctx context.Context,
	reportName string,
) ([]models.SheduleUnit, error) {
	const query = `select c.cron as cron, c.name as name from report_crons rc
	left join crons c on c.id = rc.cron_id
	left join reports r on rc.report_id = r.id
	where c.is_active = TRUE and r.name = $1`

	var crons []models.SheduleUnit

	err := r.db.SelectContext(ctx, &crons, query, reportName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []models.SheduleUnit{}, nil
		}

		return nil, err
	}

	return crons, nil
}

func (o *ReportRepository) loadReportByName(
	ctx context.Context,
	reportName string,
) (*report, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("orchestrator load reports: %w", ctx.Err())
	}

	const query = `select r.id, r.name, r.title, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id
where r.name =$1
Limit 1
;
`

	var rp report

	err := o.db.GetContext(ctx, &rp, query, reportName)
	if err != nil {
		return nil, err
	}

	return &rp, nil
}

func (o *ReportRepository) loadQueriesByReportID(
	ctx context.Context,
	reportID int,
) ([]card, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("orchestrator load queries by report id: %w", ctx.Err())
	}

	const query = `select q.card_uuid, q.title
from report_queries rq
join queries q on q.id = rq.query_id
where rq.report_id = $1
;

`

	var crds []card

	err := o.db.SelectContext(ctx, &crds, query, reportID)
	if err != nil {
		return nil, err
	}

	return crds, nil
}

func (o *ReportRepository) loadRecipients(
	ctx context.Context,
	reportID int,
) ([]recipient, error) {
	const query = `
select
    r.name,
    r.config,
    r.remote_path,
    r.thread_id,
    r.email_id,
    r.type,
    r.need_delete_after_end_of_day,

	e.dest,
	e.copy,
	e.subject,
	e.body,

    c.chat_id,
    c.title,
    c.type as chat_type,
    c.description,
    c.is_active,
    coalesce(c.ch_type, '') as ch_type
from reports_recipients rr
join recipients r on r.id = rr.recipient_id
left join chats c on c.id = r.chat_id
left join email_templates e  on e.id = r.email_id
where rr.report_id = $1
;

`

	var rcpt []recipient

	err := o.db.SelectContext(ctx, &rcpt, query, reportID)
	if err != nil {
		return nil, err
	}

	return rcpt, nil
}

func (o *ReportRepository) loadExports(
	ctx context.Context,
	reportID int,
) ([]export, error) {
	const query = `
select ef.format, re.file_name, t.id, t.title, t.type, t.template_text, re.sort_order
from reports_export re
join export_formats ef on ef.id = re.format_id
left join report_templates rt on rt.report_id = re.report_id
left join templates t on t.id = rt.template_id
where re.report_id = $1
;

`

	var exprt []export

	err := o.db.SelectContext(ctx, &exprt, query, reportID)
	if err != nil {
		return nil, err
	}

	return exprt, nil
}

func (o *ReportRepository) getReportByID(
	ctx context.Context,
	r report,
) (*models.Report, error) {
	crds, err := o.loadQueriesByReportID(ctx, r.ID)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading queries for report", slog.Any("error", err))

		return nil, err
	}

	mCrds := mapCardsToModels(crds...)

	rcpts, err := o.loadRecipients(ctx, r.ID)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading recipients for report", slog.Any("error", err))

		return nil, err
	}

	mRcpts := mapRecipientsToModel(rcpts...)

	exptrs, err := o.loadExports(ctx, r.ID)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading exports for report", slog.Any("error", err))

		return nil, err
	}

	mExprt, err := mapExportsToModel(exptrs...)
	if err != nil {
		o.log.ErrorContext(ctx, "error with map exports", slog.Any("error", err))

		return nil, err
	}

	return &models.Report{
		Name:       r.Name,
		Title:      r.Title,
		Queries:    mCrds,
		Recipients: mRcpts,
		Exports:    mExprt,
		Evaluation: r.Expr,
	}, nil
}

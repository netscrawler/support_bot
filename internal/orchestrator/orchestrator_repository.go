package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	models "support_bot/internal/models/report"

	"github.com/jmoiron/sqlx"
)

type OrchestratorRepository struct {
	db *sqlx.DB

	log *slog.Logger
}

func NewRepository(db *sqlx.DB, log *slog.Logger) *OrchestratorRepository {
	l := log.With("module", "orchestrator_repository")

	return &OrchestratorRepository{
		db:  db,
		log: l,
	}
}

func (o *OrchestratorRepository) Load(ctx context.Context) ([]models.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("orchestrator load card: %w", ctx.Err())
	}
	o.log.DebugContext(ctx, "start loading with tx")

	tx, err := o.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	})
	if err != nil {
		o.log.ErrorContext(ctx, "transaction start failed", slog.Any("error", err))

		return nil, err
	}
	defer tx.Rollback()

	o.log.DebugContext(ctx, "start loading reports")

	rpts, err := o.loadReports(ctx, tx)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading reports", slog.Any("error", err))

		return nil, err
	}

	reports := make([]models.Report, 0, len(rpts))
	for _, r := range rpts {
		rpt, err := o.getReportByID(ctx, r, tx)
		if err != nil {
			continue
		}

		reports = append(reports, *rpt)
	}

	return reports, nil
}

func (o *OrchestratorRepository) LoadByEvent(
	ctx context.Context,
	event string,
) (*models.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("orchestrator load card: %w", ctx.Err())
	}

	o.log.DebugContext(ctx, "start loading with tx")
	defer o.log.DebugContext(ctx, "finish loading")

	tx, err := o.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	})
	if err != nil {
		o.log.ErrorContext(ctx, "transaction start failed", slog.Any("error", err))

		return nil, err
	}
	defer tx.Rollback()

	o.log.DebugContext(ctx, "start loading reports")

	rpt, err := o.loadReportByName(ctx, event, tx)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading reports", slog.Any("error", err))

		return nil, err
	}

	r, err := o.getReportByID(ctx, rpt, tx)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (o *OrchestratorRepository) loadReports(ctx context.Context, tx *sqlx.Tx) ([]report, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("orchestrator load reports: %w", ctx.Err())
	}
	const query = `select r.id, r.name, r.title, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id
where r.active = true
;

`

	var rp []report

	err := tx.SelectContext(ctx, &rp, query)
	if err != nil {
		return nil, err
	}

	return rp, nil
}

func (o *OrchestratorRepository) loadReportByName(
	ctx context.Context,
	name string, tx *sqlx.Tx,
) (report, error) {
	if err := ctx.Err(); err != nil {
		return report{}, fmt.Errorf("orchestrator load report by name: %w", ctx.Err())
	}
	const query = `select r.id, r.name, r.title, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id
where r.name = $1 and active = true
;

`

	var rp report

	err := tx.GetContext(ctx, &rp, query, name)
	if err != nil {
		return report{}, err
	}

	return rp, nil
}

func (o *OrchestratorRepository) loadQueriesByReportID(
	ctx context.Context,
	reportID int, tx *sqlx.Tx,
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

	err := tx.SelectContext(ctx, &crds, query, reportID)
	if err != nil {
		return nil, err
	}

	return crds, nil
}

func (o *OrchestratorRepository) loadRecipients(
	ctx context.Context,
	reportID int, tx *sqlx.Tx,
) ([]recipient, error) {
	const query = `
select
    r.name,
    r.config,
    r.remote_path,
    r.thread_id,
    r.email_id,
    r.type,

	e.dest,
	e.copy,
	e.subject,
	e.body,

    c.chat_id,
    c.title,
    c.type as chat_type,
    c.description,
    c.is_active
from reports_recipients rr
join recipients r on r.id = rr.recipient_id
left join chats c on c.id = r.chat_id
left join email_templates e  on e.id = r.email_id
where rr.report_id = $1
;

`

	var rcpt []recipient

	err := tx.SelectContext(ctx, &rcpt, query, reportID)
	if err != nil {
		return nil, err
	}

	return rcpt, nil
}

func (o *OrchestratorRepository) loadExports(
	ctx context.Context,
	reportID int,
	tx *sqlx.Tx,
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

	err := tx.SelectContext(ctx, &exprt, query, reportID)
	if err != nil {
		return nil, err
	}

	return exprt, nil
}

func (o *OrchestratorRepository) getReportByID(
	ctx context.Context,
	r report,
	tx *sqlx.Tx,
) (*models.Report, error) {
	crds, err := o.loadQueriesByReportID(ctx, r.ID, tx)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading queries for report", slog.Any("error", err))

		return nil, err
	}

	mCrds := mapCardsToModels(crds...)

	rcpts, err := o.loadRecipients(ctx, r.ID, tx)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading recipients for report", slog.Any("error", err))

		return nil, err
	}

	mRcpts := mapRecipientsToModel(rcpts...)

	exptrs, err := o.loadExports(ctx, r.ID, tx)
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

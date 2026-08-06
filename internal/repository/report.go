package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"support_bot/internal/models"
	"support_bot/internal/pkg/uow"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB

	log *slog.Logger
}

func NewRepository(db *sqlx.DB, log *slog.Logger) *Repository {
	l := log.With("module", "report_repository")

	return &Repository{
		db:  db,
		log: l,
	}
}

func (r *Repository) NewUOW(ctx context.Context) (uow.UOW, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		r.log.ErrorContext(ctx, "failed to begin transaction", slog.Any("error", err))

		return nil, err
	}

	u := uow.NewUOW(tx)

	return u, nil
}

func (r *Repository) FindReportByName(ctx context.Context, name string, u uow.UOW) (bool, error) {
	var count int64

	err := u.GetContext(ctx, &count, "SELECT COUNT(*) FROM reports WHERE name = $1", name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("failed to find report: %w", err)
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

func (r *Repository) FindEvaluationByRule(
	ctx context.Context,
	ev string,
	u uow.UOW,
) (int64, error) {
	var id int64

	err := u.GetContext(ctx, &id, "SELECT id FROM evaluate WHERE expr = $1", ev)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find evaluation: %w", err)
	}

	return id, nil
}

func (r *Repository) CreateEvaluation(ctx context.Context, expr string, tx uow.UOW) (int64, error) {
	const query = "insert into evaluate(expr) values ($1) returning id"

	var id int64

	err := tx.GetContext(ctx, &id, query, expr)
	if err != nil {
		return 0, fmt.Errorf("failed to create evaluation: %w", err)
	}

	return id, nil
}

func (r *Repository) CreatePipeline(
	ctx context.Context,
	pipe json.RawMessage,
	tx uow.UOW,
) (int64, error) {
	const query = "insert into pipelines(pipeline) values ($1) returning id"

	var id int64

	err := tx.GetContext(ctx, &id, query, pipe)
	if err != nil {
		return 0, fmt.Errorf("failed to create pipeline: %w", err)
	}

	return id, nil
}

func (r *Repository) FindCardByUUIDAndTitle(
	ctx context.Context,
	uuid, title string,
	u uow.UOW,
) (int64, error) {
	const query = `select id from queries where card_uuid = $1 and title = $2`

	var id int64

	err := u.GetContext(ctx, &id, query, uuid, title)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find card: %w", err)
	}

	return id, nil
}

func (r *Repository) FindCardByName(ctx context.Context, name string, u uow.UOW) (int64, error) {
	const query = `select id from queries where title = $1`

	var id int64

	err := u.GetContext(ctx, &id, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find card: %w", err)
	}

	return id, nil
}

func (r *Repository) CreateQuery(ctx context.Context, card models.Card, u uow.UOW) (int64, error) {
	const query = "insert into queries( card_uuid, title, q_type, params) values ($1, $2, $3, $4) returning id"

	var id int64

	var p any
	if card.Params == nil {
		p = "{}"
	} else {
		p = card.Params
	}

	err := u.GetContext(ctx, &id, query, card.CardUUID, card.Title, card.Type, p)
	if err != nil {
		return 0, fmt.Errorf("failed to create card: %w", err)
	}

	return id, nil
}

func (r *Repository) FindRecipientFromName(
	ctx context.Context,
	name string,
	u uow.UOW,
) (int64, error) {
	var id int64

	err := u.GetContext(ctx, &id, "select id from recipients where name = $1", name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find recipient: %w", err)
	}

	return id, nil
}

type RecipientDBO struct {
	Name                    string
	Cfg                     json.RawMessage
	RemotePath              *string
	ChatID                  *int64
	ThreadID                *int
	EmailID                 *int64
	Type                    string
	NeedDeleteAfterEndOfDay bool
}

type ReportDBO struct {
	Name         string
	Title        string
	EvalID       int64
	PipelineID   *int64
	AccessFromLK bool
	Active       bool
}

func (r *Repository) CreateRecipient(
	ctx context.Context,
	recipient RecipientDBO,
	u uow.UOW,
) (int64, error) {
	const query = "insert into recipients(name, config, remote_path, chat_id, thread_id, email_id, type, need_delete_after_end_of_day) values ($1, '{}', $2, $3, $4, $5, $6, $7) returning id"

	var id int64

	err := u.GetContext(
		ctx,
		&id,
		query,
		recipient.Name,
		recipient.RemotePath,
		recipient.ChatID,
		recipient.ThreadID,
		recipient.EmailID,
		recipient.Type,
		recipient.NeedDeleteAfterEndOfDay,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create recipient: %w", err)
	}

	return id, nil
}

func (r *Repository) FindChatByID(ctx context.Context, chatID int64, u uow.UOW) (int64, error) {
	var id int64

	err := u.GetContext(ctx, &id, "select id from chats where chat_id = $1", chatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find chat: %w", err)
	}

	return id, nil
}

func (r *Repository) CreateChat(ctx context.Context, chat models.Chat, u uow.UOW) (int64, error) {
	const query = "insert into chats( chat_id, title, type, description, is_active, ch_type) values ($1, $2, $3, $4, $5, $6) returning id"

	var id int64

	err := u.GetContext(
		ctx,
		&id,
		query,
		chat.ChatID,
		chat.Title,
		chat.Type,
		chat.Description,
		chat.IsActive,
		chat.ChType,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create chat: %w", err)
	}

	return id, nil
}

func (r *Repository) CreateEmailTemplate(
	ctx context.Context,
	email models.EmailTemplate,
	u uow.UOW,
) (int64, error) {
	const query = "insert into email_templates( dest, copy, subject, body) values ($1, $2, $3, $4) returning id"

	var id int64

	err := u.GetContext(
		ctx,
		&id,
		query,
		email.Dest,
		email.Copy,
		email.Subject,
		email.Body,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create email template: %w", err)
	}

	return id, nil
}

func (r *Repository) CreateReport(ctx context.Context, report ReportDBO, u uow.UOW) (int64, error) {
	const query = `insert into reports(name, title, eval_id, pipeline_id, access_from_lk, active) 
                   values ($1, $2, $3, $4, $5, $6) returning id`

	var id int64

	err := u.GetContext(
		ctx,
		&id,
		query,
		report.Name,
		report.Title,
		report.EvalID,
		report.PipelineID,
		report.AccessFromLK,
		report.Active,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create report: %w", err)
	}

	return id, nil
}

func (r *Repository) LinkQueryToReport(
	ctx context.Context,
	reportID, queryID int64,
	u uow.UOW,
) error {
	const query = "insert into report_queries(report_id, query_id) values ($1, $2)"

	_, err := u.ExecContext(ctx, query, reportID, queryID)
	if err != nil {
		return fmt.Errorf("failed to link query to report: %w", err)
	}

	return nil
}

func (r *Repository) LinkRecipientToReport(
	ctx context.Context,
	reportID, recipientID int64,
	u uow.UOW,
) error {
	const query = "insert into reports_recipients(report_id, recipient_id) values ($1, $2)"

	_, err := u.ExecContext(ctx, query, reportID, recipientID)
	if err != nil {
		return fmt.Errorf("failed to link recipient to report: %w", err)
	}

	return nil
}

func (r *Repository) FindExportFormat(
	ctx context.Context,
	format string,
	u uow.UOW,
) (int64, error) {
	var id int64

	err := u.GetContext(ctx, &id, "select id from export_formats where format = $1", format)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find export format: %w", err)
	}

	return id, nil
}

func (r *Repository) CreateExportFormat(
	ctx context.Context,
	format string,
	u uow.UOW,
) (int64, error) {
	var id int64

	err := u.GetContext(
		ctx,
		&id,
		"insert into export_formats(format) values ($1) returning id",
		format,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create export format: %w", err)
	}

	return id, nil
}

func (r *Repository) LinkExportToReport(
	ctx context.Context,
	reportID, formatID int64,
	fileName *string,
	sortOrder json.RawMessage,
	u uow.UOW,
) error {
	const query = "insert into reports_export(report_id, format_id, file_name, sort_order) values ($1, $2, $3, $4)"

	_, err := u.ExecContext(ctx, query, reportID, formatID, fileName, sortOrder)
	if err != nil {
		return fmt.Errorf("failed to link export to report: %w", err)
	}

	return nil
}

func (r *Repository) LinkTemplateToReport(
	ctx context.Context,
	reportID, templateID int64,
	u uow.UOW,
) error {
	const query = "insert into report_templates(report_id, template_id) values ($1, $2)"

	_, err := u.ExecContext(ctx, query, reportID, templateID)
	if err != nil {
		return fmt.Errorf("failed to link template to report: %w", err)
	}

	return nil
}

func (r *Repository) CreateTemplate(
	ctx context.Context,
	tmpl models.Template,
	u uow.UOW,
) (int64, error) {
	const query = "insert into templates(template_text, title, type) values ($1, $2, $3) returning id"

	var id int64

	err := u.GetContext(ctx, &id, query, tmpl.TemplateText, tmpl.Title, tmpl.Type)
	if err != nil {
		return 0, fmt.Errorf("failed to create template: %w", err)
	}

	return id, nil
}

func (r *Repository) FindTemplateByTitleAndType(
	ctx context.Context,
	title, tType string,
	u uow.UOW,
) (int64, error) {
	var id int64

	err := u.GetContext(
		ctx,
		&id,
		"select id from templates where title = $1 and type = $2",
		title,
		tType,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find template: %w", err)
	}

	return id, nil
}

func (r *Repository) FindCronByNameAndExpr(
	ctx context.Context,
	name, cron string,
	u uow.UOW,
) (int64, error) {
	var id int64

	err := u.GetContext(ctx, &id, "select id from crons where name = $1 and cron = $2", name, cron)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, models.ErrNotFound
		}

		return 0, fmt.Errorf("failed to find cron: %w", err)
	}

	return id, nil
}

func (r *Repository) CreateCron(ctx context.Context, cron models.Cron, u uow.UOW) (int64, error) {
	const query = "insert into crons(cron, name, description, is_active, event_type) values ($1, $2, $3, $4, $5) returning id"

	var id int64

	err := u.GetContext(
		ctx,
		&id,
		query,
		cron.Cron,
		cron.Name,
		cron.Description,
		cron.IsActive,
		cron.EventType,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create cron: %w", err)
	}

	return id, nil
}

func (r *Repository) LinkCronToReport(
	ctx context.Context,
	reportID, cronID int64,
	u uow.UOW,
) error {
	const query = "insert into report_crons(report_id, cron_id) values ($1, $2)"

	_, err := u.ExecContext(ctx, query, reportID, cronID)
	if err != nil {
		return fmt.Errorf("failed to link cron to report: %w", err)
	}

	return nil
}

func (r *Repository) Load(ctx context.Context) ([]models.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("repository load: %w", ctx.Err())
	}

	r.log.DebugContext(ctx, "start loading all reports")

	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	})
	if err != nil {
		r.log.ErrorContext(ctx, "transaction start failed", slog.Any("error", err))

		return nil, err
	}
	defer tx.Rollback()

	u := uow.NewUOW(tx)

	rpts, err := r.loadReports(ctx, u)
	if err != nil {
		r.log.ErrorContext(ctx, "error loading reports", slog.Any("error", err))

		return nil, err
	}

	reports := make([]models.Report, 0, len(rpts))
	for _, rp := range rpts {
		rpt, err := r.getReportByID(ctx, rp, u)
		if err != nil {
			r.log.ErrorContext(
				ctx,
				"error getting full report",
				slog.Any("report_name", rp.Name),
				slog.Any("error", err),
			)

			continue
		}

		reports = append(reports, *rpt)
	}

	return reports, nil
}

func (r *Repository) loadReports(ctx context.Context, u uow.UOW) ([]reportLoad, error) {
	const query = `select r.id, r.name, r.title, r.active, r.access_from_lk, r.pipeline_id, e.expr as evaluation
from reports r
left join evaluate e on e.id = r.eval_id`

	var rp []reportLoad

	err := u.SelectContext(ctx, &rp, query)
	if err != nil {
		return nil, err
	}

	return rp, nil
}

func (r *Repository) getReportByID(
	ctx context.Context,
	rp reportLoad,
	u uow.UOW,
) (*models.Report, error) {
	// Queries
	const queryQueries = `select q.card_uuid, q.title, q.q_type, q.params
from report_queries rq
join queries q on q.id = rq.query_id
where rq.report_id = $1`

	var crds []cardLoad

	err := u.SelectContext(ctx, &crds, queryQueries, rp.ID)
	if err != nil {
		return nil, fmt.Errorf("load queries: %w", err)
	}

	// Recipients
	const queryRecipients = `
select
    rc.name,
    rc.config,
    rc.remote_path,
    rc.thread_id,
    rc.email_id,
    rc.type,
    rc.need_delete_after_end_of_day,

	e.dest,
	e.copy,
	e.subject,
	e.body,

    c.chat_id,
    c.title as chat_title,
    c.type as chat_type,
    c.description as chat_description,
    c.is_active as chat_is_active,
    c.ch_type as chat_ch_type
from reports_recipients rr
join recipients rc on rc.id = rr.recipient_id
left join chats c on c.id = rc.chat_id
left join email_templates e  on e.id = rc.email_id
where rr.report_id = $1`

	var rcpts []recipientLoad

	err = u.SelectContext(ctx, &rcpts, queryRecipients, rp.ID)
	if err != nil {
		return nil, fmt.Errorf("load recipients: %w", err)
	}

	// Exports
	const queryExports = `
select ef.format, re.file_name, t.id as template_id, t.title as template_title, t.type as template_type, t.template_text, re.sort_order
from reports_export re
join export_formats ef on ef.id = re.format_id
left join report_templates rt on rt.report_id = re.report_id
left join templates t on t.id = rt.template_id
where re.report_id = $1`

	var exprts []exportLoad

	err = u.SelectContext(ctx, &exprts, queryExports, rp.ID)
	if err != nil {
		return nil, fmt.Errorf("load exports: %w", err)
	}

	// Crons
	const queryCrons = `
select c.cron, c.name, c.description, c.is_active, c.event_type
from report_crons rc
join crons c on c.id = rc.cron_id
where rc.report_id = $1`

	var crns []cronLoad

	err = u.SelectContext(ctx, &crns, queryCrons, rp.ID)
	if err != nil {
		return nil, fmt.Errorf("load crons: %w", err)
	}

	// Pipeline
	var pipe *models.Pipeline

	if rp.PipelineID != nil {
		const queryPipeline = `select pipeline from pipelines where id = $1`

		var pipeData json.RawMessage

		err = u.GetContext(ctx, &pipeData, queryPipeline, *rp.PipelineID)
		if err != nil {
			return nil, fmt.Errorf("load pipeline: %w", err)
		}

		err = json.Unmarshal(pipeData, &pipe)
		if err != nil {
			return nil, fmt.Errorf("unmarshal pipeline: %w", err)
		}
	}

	mCrds := mapCardsToModels(crds...)
	mRcpts := mapRecipientsToModels(rcpts...)

	mExprts, err := mapExportsToModels(exprts...)
	if err != nil {
		return nil, fmt.Errorf("map exports: %w", err)
	}

	mCrns := mapCronsToModels(crns...)

	return &models.Report{
		Name:         rp.Name,
		Title:        rp.Title,
		Queries:      mCrds,
		Recipients:   mRcpts,
		Exports:      mExprts,
		Pipeline:     pipe,
		Evaluation:   rp.Evaluation,
		Active:       rp.Active,
		AccessFromLK: rp.AccessFromLK,
		Crons:        mCrns,
	}, nil
}

type reportLoad struct {
	ID           int    `db:"id"`
	Name         string `db:"name"`
	Title        string `db:"title"`
	Active       bool   `db:"active"`
	AccessFromLK bool   `db:"access_from_lk"`
	Evaluation   string `db:"evaluation"`
	PipelineID   *int64 `db:"pipeline_id"`
}

type cardLoad struct {
	CardUUID string          `db:"card_uuid"`
	Title    string          `db:"title"`
	Params   json.RawMessage `db:"params"`
	Type     string          `db:"q_type"`
}

type recipientLoad struct {
	Name string `db:"name"`

	Config json.RawMessage `db:"config"`

	RemotePath *string `db:"remote_path"`

	EmailID *int    `db:"email_id"`
	Dest    *string `db:"dest"`
	Copy    *string `db:"copy"`
	Subject *string `db:"subject"`
	Body    *string `db:"body"`
	Type    string  `db:"type"`

	ChatID          *int64  `db:"chat_id"`
	ThreadID        *int    `db:"thread_id"`
	ChatTitle       *string `db:"chat_title"`
	ChatType        *string `db:"chat_type"`
	ChatDescription *string `db:"chat_description"`
	ChatIsActive    *bool   `db:"chat_is_active"`
	ChatChType      *string `db:"chat_ch_type"`

	NeedDeleteAfterEndOfDay *bool `db:"need_delete_after_end_of_day"`
}

type exportLoad struct {
	Format   string  `db:"format"`
	FileName *string `db:"file_name"`

	TemplateID    *int            `db:"template_id"`
	TemplateTitle *string         `db:"template_title"`
	TemplateType  *string         `db:"template_type"`
	TemplateText  *string         `db:"template_text"`
	Order         json.RawMessage `db:"sort_order"`
}

type cronLoad struct {
	Cron        string `db:"cron"`
	Name        string `db:"name"`
	Description string `db:"description"`
	IsActive    bool   `db:"is_active"`
	EventType   int    `db:"event_type"`
}

func mapCardsToModels(c ...cardLoad) []models.Card {
	var crds []models.Card

	for _, crd := range c {
		crds = append(crds, models.Card{
			CardUUID:  crd.CardUUID,
			Title:     crd.Title,
			Type:      crd.Type,
			RawParams: crd.Params,
		})
	}

	return crds
}

func mapRecipientsToModels(r ...recipientLoad) []models.Recipient {
	var rcpts []models.Recipient

	for _, rcpt := range r {
		var c *models.Chat

		if rcpt.ChatID != nil {
			c = &models.Chat{
				ChatID:      *rcpt.ChatID,
				Title:       rcpt.ChatTitle,
				Type:        deref(rcpt.ChatType),
				Description: rcpt.ChatDescription,
				IsActive:    deref(rcpt.ChatIsActive),
				ChType:      deref(rcpt.ChatChType),
			}
		}

		var e *models.EmailTemplate

		var dest []string

		if rcpt.Dest != nil {
			dest = pqArrayToArray(*rcpt.Dest)
		}

		var rCopy []string

		if rcpt.Copy != nil {
			rCopy = pqArrayToArray(*rcpt.Copy)
		}

		if rcpt.EmailID != nil {
			e = &models.EmailTemplate{
				Dest:    dest,
				Copy:    rCopy,
				Subject: deref(rcpt.Subject),
				Body:    rcpt.Body,
			}
		}

		needDeleteAfterEndOfDay := false

		if rcpt.NeedDeleteAfterEndOfDay != nil {
			needDeleteAfterEndOfDay = *rcpt.NeedDeleteAfterEndOfDay
		}

		rcpts = append(rcpts, models.Recipient{
			Name:                    rcpt.Name,
			Config:                  rcpt.Config,
			RemotePath:              rcpt.RemotePath,
			Chat:                    c,
			ThreadID:                rcpt.ThreadID,
			Email:                   e,
			Type:                    models.RecipientType(rcpt.Type),
			NeedDeleteAfterEndOfDay: needDeleteAfterEndOfDay,
		})
	}

	return rcpts
}

func deref[T any](t *T) T {
	if t != nil {
		return *t
	}

	var zero T

	return zero
}

func pqArrayToArray(arr string) []string {
	a := strings.TrimLeft(arr, "{")
	a = strings.TrimRight(a, "}")
	if a == "" {
		return nil
	}

	return strings.Split(a, ",")
}

func mapExportsToModels(e ...exportLoad) ([]models.Export, error) {
	var exprts []models.Export

	for _, exp := range e {
		var t *models.Template

		if exp.TemplateID != nil {
			t = &models.Template{
				ID:           *exp.TemplateID,
				Title:        deref(exp.TemplateTitle),
				Type:         deref(exp.TemplateType),
				TemplateText: deref(exp.TemplateText),
			}
		}

		var order map[string][]string

		if exp.Order != nil {
			err := json.Unmarshal(exp.Order, &order)
			if err != nil {
				return nil, err
			}
		}

		exprts = append(exprts, models.Export{
			Format:   exp.Format,
			Template: t,
			FileName: exp.FileName,
			Order:    order,
		})
	}

	return exprts, nil
}

func mapCronsToModels(c ...cronLoad) []models.Cron {
	var crns []models.Cron

	for _, crn := range c {
		crns = append(crns, models.Cron{
			Name:        crn.Name,
			Cron:        crn.Cron,
			Description: crn.Description,
			IsActive:    crn.IsActive,
			EventType:   crn.EventType,
		})
	}

	return crns
}

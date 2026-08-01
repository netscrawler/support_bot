package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"support_bot/internal/models"
	"support_bot/internal/pkg/uow"
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

	err := u.GetContext(ctx, &id, query, card.CardUUID, card.Title, card.Type, card.Params)
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
	const query = "insert into recipients(name, config, remote_path, chat_id, thread_id, email_id, type, need_delete_after_end_of_day) values ($1, $2, $3, $4, $5, $6, $7, $8) returning id"

	var id int64

	err := u.GetContext(
		ctx,
		&id,
		query,
		recipient.Name,
		recipient.Cfg,
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

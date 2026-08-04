package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"support_bot/internal/models"
	"support_bot/internal/pkg/uow"
	"support_bot/internal/repository"
)

type ReportManager struct {
	repo *repository.Repository
	val  *ReportValidation
	log  *slog.Logger
}

func NewReportManager(
	repo *repository.Repository,
	val *ReportValidation,
	log *slog.Logger,
) *ReportManager {
	return &ReportManager{
		repo: repo,
		val:  val,
		log:  log.With("module", "report_manager"),
	}
}

func (m *ReportManager) Load(ctx context.Context) ([]models.Report, error) {
	m.log.InfoContext(ctx, "Loading all reports")

	return m.repo.Load(ctx)
}

// Create creates a report and all its components in a single transaction.
func (m *ReportManager) Create(ctx context.Context, repr io.Reader) error {
	var rep models.Report
	err := json.NewDecoder(repr).Decode(&rep)
	if err != nil {
		m.log.ErrorContext(
			ctx,
			"Failed to unmarshal JSON",
			slog.Any("error", err),
		)
		return err
	}
	m.log.InfoContext(ctx, "Creating report", slog.String("name", rep.Name))

	if err := m.val.Validate(ctx, rep); err != nil {
		return fmt.Errorf("validate report: %w", err)
	}

	u, err := m.repo.NewUOW(ctx)
	if err != nil {
		return fmt.Errorf("create UOW: %w", err)
	}

	defer func() {
		_ = u.Rollback()
	}()

	exists, err := m.repo.FindReportByName(ctx, rep.Name, u)
	if err != nil {
		return fmt.Errorf("find report by name: %w", err)
	}

	if exists {
		return fmt.Errorf("report with name %q already exists", rep.Name)
	}

	// 1. Pipeline
	var pipelineID *int64

	if rep.Pipeline != nil {
		m.log.InfoContext(ctx, "Creating pipeline")

		id, err := m.createPipeline(ctx, rep.Pipeline, u)
		if err != nil {
			return fmt.Errorf("create pipeline: %w", err)
		}

		pipelineID = &id
	}

	// 2. Evaluation
	evalID, err := m.getOrCreateEvaluation(ctx, rep.Evaluation, u)
	if err != nil {
		return fmt.Errorf("process evaluation: %w", err)
	}

	// 3. Report
	reportID, err := m.repo.CreateReport(ctx, repository.ReportDBO{
		Name:         rep.Name,
		Title:        rep.Title,
		EvalID:       evalID,
		PipelineID:   pipelineID,
		AccessFromLK: rep.AccessFromLK,
		Active:       rep.Active,
	}, u)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}

	// 4. Queries
	m.log.InfoContext(ctx, "Processing queries")

	for _, query := range rep.Queries {
		m.log.InfoContext(ctx, "Creating/Using query", slog.String("title", query.Title))

		qID, err := m.getOrCreateQuery(ctx, query, u)
		if err != nil {
			return fmt.Errorf("process query %q: %w", query.Title, err)
		}

		if err := m.repo.LinkQueryToReport(ctx, reportID, qID, u); err != nil {
			return fmt.Errorf("link query %d: %w", qID, err)
		}
	}

	// 5. Exports and Templates
	m.log.InfoContext(ctx, "Processing exports")

	for _, export := range rep.Exports {
		m.log.InfoContext(ctx, "Creating export", slog.String("format", string(export.Format)))

		formatID, err := m.getOrCreateExportFormat(ctx, string(export.Format), u)
		if err != nil {
			return fmt.Errorf("process export format %q: %w", export.Format, err)
		}

		sortOrder, err := json.Marshal(export.Order)
		if err != nil {
			return fmt.Errorf("marshal export order: %w", err)
		}

		err = m.repo.LinkExportToReport(ctx, reportID, formatID, export.FileName, sortOrder, u)
		if err != nil {
			return fmt.Errorf("link export: %w", err)
		}

		if export.Template != nil {
			m.log.InfoContext(
				ctx,
				"Creating/Using template",
				slog.String("title", export.Template.Title),
			)

			tmplID, err := m.getOrCreateTemplate(ctx, *export.Template, u)
			if err != nil {
				return fmt.Errorf("process template %q: %w", export.Template.Title, err)
			}

			if err := m.repo.LinkTemplateToReport(ctx, reportID, tmplID, u); err != nil {
				return fmt.Errorf("link template: %w", err)
			}
		}
	}

	// 6. Crons
	m.log.InfoContext(ctx, "Processing crons")

	for _, cron := range rep.Crons {
		m.log.InfoContext(
			ctx,
			"Creating/Using cron",
			slog.String("name", cron.Name),
			slog.String("cron", cron.Cron),
		)

		cID, err := m.getOrCreateCron(ctx, cron, u)
		if err != nil {
			return fmt.Errorf("process cron %q: %w", cron.Name, err)
		}

		if err := m.repo.LinkCronToReport(ctx, reportID, cID, u); err != nil {
			return fmt.Errorf("link cron %d: %w", cID, err)
		}
	}

	// 7. Recipients
	m.log.InfoContext(ctx, "Processing recipients")

	for _, recipient := range rep.Recipients {
		m.log.InfoContext(ctx, "Creating/Using recipient", slog.String("name", recipient.Name))

		rID, err := m.getOrCreateRecipient(ctx, recipient, u)
		if err != nil {
			return fmt.Errorf("process recipient %q: %w", recipient.Name, err)
		}

		if err := m.repo.LinkRecipientToReport(ctx, reportID, rID, u); err != nil {
			return fmt.Errorf("link recipient %d: %w", rID, err)
		}
	}

	if err := u.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	m.log.InfoContext(ctx, "Report successfully created", slog.String("name", rep.Name))

	return nil
}

func (m *ReportManager) getOrCreateEvaluation(
	ctx context.Context,
	eval string,
	u uow.UOW,
) (int64, error) {
	id, err := m.repo.FindEvaluationByRule(ctx, eval, u)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("find evaluation: %w", err)
	}

	return m.repo.CreateEvaluation(ctx, eval, u)
}

func (m *ReportManager) createPipeline(
	ctx context.Context,
	pipe *models.Pipeline,
	u uow.UOW,
) (int64, error) {
	pipeBytes, err := json.Marshal(pipe)
	if err != nil {
		return 0, fmt.Errorf("marshal pipeline: %w", err)
	}

	return m.repo.CreatePipeline(ctx, pipeBytes, u)
}

func (m *ReportManager) getOrCreateQuery(
	ctx context.Context,
	query models.Card,
	u uow.UOW,
) (int64, error) {
	id, err := m.repo.FindCardByUUIDAndTitle(ctx, query.CardUUID, query.Title, u)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("find query: %w", err)
	}

	return m.repo.CreateQuery(ctx, query, u)
}

func (m *ReportManager) getOrCreateExportFormat(
	ctx context.Context,
	format string,
	u uow.UOW,
) (int64, error) {
	id, err := m.repo.FindExportFormat(ctx, format, u)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("find export format: %w", err)
	}

	return m.repo.CreateExportFormat(ctx, format, u)
}

func (m *ReportManager) getOrCreateTemplate(
	ctx context.Context,
	tmpl models.Template,
	u uow.UOW,
) (int64, error) {
	id, err := m.repo.FindTemplateByTitleAndType(ctx, tmpl.Title, tmpl.Type, u)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("find template: %w", err)
	}

	return m.repo.CreateTemplate(ctx, tmpl, u)
}

func (m *ReportManager) getOrCreateCron(
	ctx context.Context,
	cron models.Cron,
	u uow.UOW,
) (int64, error) {
	id, err := m.repo.FindCronByNameAndExpr(ctx, cron.Name, cron.Cron, u)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("find cron: %w", err)
	}

	return m.repo.CreateCron(ctx, cron, u)
}

func (m *ReportManager) getOrCreateRecipient(
	ctx context.Context,
	recipient models.Recipient,
	u uow.UOW,
) (int64, error) {
	id, err := m.repo.FindRecipientFromName(ctx, recipient.Name, u)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("find recipient: %w", err)
	}

	return m.createRecipient(ctx, recipient, u)
}

func (m *ReportManager) createRecipient(
	ctx context.Context,
	recipient models.Recipient,
	u uow.UOW,
) (int64, error) {
	var (
		chatID  *int64
		emailID *int64
	)

	if recipient.Chat != nil {
		id, err := m.getOrCreateChat(ctx, *recipient.Chat, u)
		if err != nil {
			return 0, err
		}

		chatID = &id
	}

	if recipient.Email != nil {
		id, err := m.repo.CreateEmailTemplate(ctx, *recipient.Email, u)
		if err != nil {
			return 0, err
		}

		emailID = &id
	}

	return m.repo.CreateRecipient(ctx, repository.RecipientDBO{
		Name:                    recipient.Name,
		Cfg:                     recipient.Config,
		RemotePath:              recipient.RemotePath,
		ChatID:                  chatID,
		ThreadID:                recipient.ThreadID,
		EmailID:                 emailID,
		Type:                    string(recipient.Type),
		NeedDeleteAfterEndOfDay: recipient.NeedDeleteAfterEndOfDay,
	}, u)
}

func (m *ReportManager) getOrCreateChat(
	ctx context.Context,
	chat models.Chat,
	u uow.UOW,
) (int64, error) {
	id, err := m.repo.FindChatByID(ctx, chat.ChatID, u)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return 0, err
	}

	return m.repo.CreateChat(ctx, chat, u)
}

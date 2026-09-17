package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportStore struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
	log  *slog.Logger
}

// ReportsPageSize — размер страницы постраничного списка отчётов.
// Общий источник истины для ReportStore.LoadPaged и
// internal/tg_bot/service.Report, которые раньше независимо дублировали
// это же значение как две отдельные константы, рискуя разойтись.
const ReportsPageSize = 5

func NewReportStore(pool *pgxpool.Pool, log *slog.Logger) *ReportStore {
	return &ReportStore{
		pool: pool,
		q:    sqlcgen.New(pool),
		log:  log.With(slog.Any("module", "store.report")),
	}
}

func (s *ReportStore) Load(ctx context.Context) ([]models.Report, error) {
	rows, err := s.q.ListReportRows(ctx)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}

	reports := make([]models.Report, 0, len(rows))

	for _, row := range rows {
		report, err := s.assemble(
			ctx,
			int64(row.ID),
			row.Name,
			row.Title,
			row.Active,
			row.AccessFromLk,
			row.PipelineID,
			derefStr(row.Evaluation),
		)
		if err != nil {
			s.log.ErrorContext(
				ctx,
				"error assembling report",
				slog.Any("report_name", row.Name),
				slog.Any("error", err),
			)

			continue
		}

		reports = append(reports, *report)
	}

	return reports, nil
}

func (s *ReportStore) LoadActive(ctx context.Context) ([]models.Report, error) {
	rows, err := s.q.ListActiveReportRows(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active reports: %w", err)
	}

	reports := make([]models.Report, 0, len(rows))

	for _, row := range rows {
		report, err := s.assemble(
			ctx,
			int64(row.ID),
			row.Name,
			row.Title,
			row.Active,
			row.AccessFromLk,
			row.PipelineID,
			derefStr(row.Evaluation),
		)
		if err != nil {
			s.log.ErrorContext(
				ctx,
				"error assembling report",
				slog.Any("report_name", row.Name),
				slog.Any("error", err),
			)

			continue
		}

		reports = append(reports, *report)
	}

	return reports, nil
}

func (s *ReportStore) GetByID(ctx context.Context, id int64) (*models.Report, error) {
	row, err := s.q.GetReportRowByID(ctx, id)
	if err != nil {
		return nil, translateNoRows(err)
	}

	return s.assemble(
		ctx,
		int64(row.ID),
		row.Name,
		row.Title,
		row.Active,
		row.AccessFromLk,
		row.PipelineID,
		derefStr(row.Evaluation),
	)
}

func (s *ReportStore) GetByName(ctx context.Context, name string) (*models.Report, error) {
	row, err := s.q.GetAnyReportRowByName(ctx, name)
	if err != nil {
		return nil, translateNoRows(err)
	}

	return s.assemble(
		ctx,
		int64(row.ID),
		row.Name,
		row.Title,
		row.Active,
		row.AccessFromLk,
		row.PipelineID,
		derefStr(row.Evaluation),
	)
}

func (s *ReportStore) LoadByEvent(
	ctx context.Context,
	event string,
	active bool,
) (*models.Report, error) {
	if active {
		row, err := s.q.GetActiveReportRowByName(ctx, event)
		if err != nil {
			return nil, translateNoRows(err)
		}

		return s.assemble(
			ctx,
			int64(row.ID),
			row.Name,
			row.Title,
			row.Active,
			row.AccessFromLk,
			row.PipelineID,
			derefStr(row.Evaluation),
		)
	}

	row, err := s.q.GetAnyReportRowByName(ctx, event)
	if err != nil {
		return nil, translateNoRows(err)
	}

	return s.assemble(
		ctx,
		int64(row.ID),
		row.Name,
		row.Title,
		row.Active,
		row.AccessFromLk,
		row.PipelineID,
		derefStr(row.Evaluation),
	)
}

func (s *ReportStore) GetByPublicID(ctx context.Context, publicID string) (*models.Report, error) {
	var id pgtype.UUID
	if err := id.Scan(publicID); err != nil {
		return nil, fmt.Errorf("parse public id: %w", err)
	}

	reportID, err := s.q.GetReportIDByPublicID(ctx, id)
	if err != nil {
		return nil, translateNoRows(err)
	}

	return s.GetByID(ctx, reportID)
}

func (s *ReportStore) LoadPaged(
	ctx context.Context,
	page int,
) ([]models.ReportForTgLK, int, error) {
	if page <= 0 {
		page = 1
	}

	total, err := s.q.CountReportsForLK(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count reports: %w", err)
	}
	if total > 0 {
		page = min(page, (int(total)+ReportsPageSize-1)/ReportsPageSize)
	}

	offset := int32(page-1) * ReportsPageSize

	rows, err := s.q.ListReportsForLK(
		ctx,
		sqlcgen.ListReportsForLKParams{Limit: ReportsPageSize, Offset: offset},
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports for lk: %w", err)
	}

	reports := make([]models.ReportForTgLK, 0, len(rows))
	for _, row := range rows {
		reports = append(
			reports,
			models.ReportForTgLK{ID: int(row.ID), Name: row.Name, Title: row.Title},
		)
	}

	return reports, int(total), nil
}

func (s *ReportStore) GetLinkedCrons(
	ctx context.Context,
	reportName string,
) ([]models.SheduleUnit, error) {
	rows, err := s.q.ListReportLinkedCrons(ctx, reportName)
	if err != nil {
		return nil, fmt.Errorf("list linked crons: %w", err)
	}

	crons := make([]models.SheduleUnit, 0, len(rows))
	for _, row := range rows {
		crons = append(
			crons,
			models.SheduleUnit{Crontab: derefStr(row.Cron), Name: derefStr(row.Name)},
		)
	}

	return crons, nil
}

// Create inserts a report and all of its dependencies (evaluation, pipeline,
// queries, exports, templates, crons, recipients) inside a single
// transaction, resolving each dependency by its natural key if it already
// exists (get-or-create) rather than duplicating it. Ports
// internal/service/report_manager.go's ReportManager.Create.
func (s *ReportStore) Create(ctx context.Context, rep models.Report) (int64, error) {
	return s.createWithPool(ctx, s.pool, rep)
}

// assemble replaces the three near-identical getFullReportModel/getReportByID
// copies previously duplicated across internal/repository,
// internal/orchestrator, and internal/tg_bot/repository — the single place
// every field (Crons, Pipeline, card Params/Type) is loaded, for every
// entry point above.
func (s *ReportStore) assemble(
	ctx context.Context,
	id int64,
	name, title string,
	active, accessFromLK bool,
	pipelineID *int64,
	evaluation string,
) (*models.Report, error) {
	cardRows, err := s.q.ListCardsByReportID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load queries: %w", err)
	}

	recipientRows, err := s.q.ListRecipientsByReportID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load recipients: %w", err)
	}

	exportRows, err := s.q.ListExportsByReportID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load exports: %w", err)
	}

	cronRows, err := s.q.ListCronsByReportID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load crons: %w", err)
	}

	var pipeline *models.Pipeline
	if pipelineID != nil {
		pipeData, err := s.q.GetPipelineByID(ctx, *pipelineID)
		if err != nil {
			return nil, translateNoRows(err)
		}

		if err := json.Unmarshal(pipeData, &pipeline); err != nil {
			return nil, fmt.Errorf("unmarshal pipeline: %w", err)
		}
	}

	return &models.Report{
		Name:         name,
		Title:        title,
		Queries:      mapCardRows(cardRows),
		Recipients:   mapRecipientRows(recipientRows),
		Exports:      s.mapExportRows(ctx, exportRows),
		Pipeline:     pipeline,
		Evaluation:   evaluation,
		Active:       active,
		AccessFromLK: accessFromLK,
		Crons:        mapCronRows(cronRows),
	}, nil
}

func mapCardRows(rows []sqlcgen.ListCardsByReportIDRow) []models.Card {
	cards := make([]models.Card, 0, len(rows))

	for _, row := range rows {
		cards = append(cards, models.Card{
			CardUUID:  row.CardUuid,
			Title:     row.Title,
			Type:      row.QType,
			RawParams: row.Params,
		})
	}

	return cards
}

func mapRecipientRows(rows []sqlcgen.ListRecipientsByReportIDRow) []models.Recipient {
	recipients := make([]models.Recipient, 0, len(rows))

	for _, row := range rows {
		var chat *models.Chat
		if row.ChatID != nil {
			chat = &models.Chat{
				ChatID:      *row.ChatID,
				Title:       row.ChatTitle,
				Type:        derefStr(row.ChatType),
				Description: row.ChatDescription,
				IsActive:    derefBool(row.ChatIsActive),
				ChType:      derefStr(row.ChatChType),
			}
		}

		var email *models.EmailTemplate
		if row.EmailID != nil {
			email = &models.EmailTemplate{
				Dest:    row.Dest,
				Copy:    row.Copy,
				Subject: derefStr(row.Subject),
				Body:    row.Body,
			}
		}

		needDelete := false
		if row.NeedDeleteAfterEndOfDay != nil {
			needDelete = *row.NeedDeleteAfterEndOfDay
		}

		recipients = append(recipients, models.Recipient{
			Name:                    row.Name,
			Config:                  row.Config,
			RemotePath:              row.RemotePath,
			Chat:                    chat,
			ThreadID:                intPtr(row.ThreadID),
			Email:                   email,
			Type:                    models.RecipientType(derefStr(row.Type)),
			NeedDeleteAfterEndOfDay: needDelete,
		})
	}

	return recipients
}

// mapExportRows — метод (а не свободная функция, как остальные map*Rows),
// потому что ему нужны logger и ctx для предупреждения о повреждённом
// sort_order, а свободной функции их взять неоткуда.
func (s *ReportStore) mapExportRows(
	ctx context.Context,
	rows []sqlcgen.ListExportsByReportIDRow,
) []models.Export {
	exports := make([]models.Export, 0, len(rows))

	for _, row := range rows {
		var tmpl *models.Template
		if row.TemplateID != nil {
			tmpl = &models.Template{
				ID:           int(*row.TemplateID),
				Title:        derefStr(row.TemplateTitle),
				Type:         derefStr(row.TemplateType),
				TemplateText: derefStr(row.TemplateText),
			}
		}

		var order map[string][]string
		if row.SortOrder != nil {
			if err := json.Unmarshal(row.SortOrder, &order); err != nil {
				s.log.WarnContext(
					ctx,
					"failed to unmarshal export sort_order",
					slog.Any("file_name", row.FileName),
					slog.Any("error", err),
				)
			}
		}

		exports = append(exports, models.Export{
			Format:   derefStr(row.Format),
			Template: tmpl,
			FileName: row.FileName,
			Order:    order,
		})
	}

	return exports
}

func mapCronRows(rows []sqlcgen.ListCronsByReportIDRow) []models.Cron {
	crons := make([]models.Cron, 0, len(rows))

	for _, row := range rows {
		crons = append(crons, models.Cron{
			Name:        row.Name,
			Cron:        row.Cron,
			Description: derefStr(row.Description),
			IsActive:    row.IsActive,
			EventType:   int(row.EventType),
		})
	}

	return crons
}

func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func derefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func intPtr(i *int32) *int {
	if i == nil {
		return nil
	}
	v := int(*i)
	return &v
}

// createWithPool is Create generalized over txBeginner so tests can drive it
// against a pgxmock pool without a real *pgxpool.Pool.
func (s *ReportStore) createWithPool(
	ctx context.Context,
	pool txBeginner,
	rep models.Report,
) (int64, error) {
	var reportID int32

	err := ExecTx(ctx, pool, func(q *sqlcgen.Queries) error {
		exists, err := q.ReportExistsByName(ctx, rep.Name)
		if err != nil {
			return fmt.Errorf("check report exists: %w", err)
		}
		if exists {
			return models.ErrAlreadyExist
		}

		var pipelineID *int64
		if rep.Pipeline != nil {
			id, err := s.createPipeline(ctx, q, rep.Pipeline)
			if err != nil {
				return fmt.Errorf("create pipeline: %w", err)
			}
			pipelineID = &id
		}

		evalID, err := s.getOrCreateEvaluation(ctx, q, rep.Evaluation)
		if err != nil {
			return fmt.Errorf("process evaluation: %w", err)
		}

		reportID, err = q.CreateReport(ctx, sqlcgen.CreateReportParams{
			Name:         rep.Name,
			Title:        rep.Title,
			EvalID:       evalID,
			PipelineID:   pipelineID,
			AccessFromLk: rep.AccessFromLK,
			Active:       rep.Active,
		})
		if err != nil {
			return fmt.Errorf("create report: %w", err)
		}

		for _, query := range rep.Queries {
			qID, err := s.getOrCreateQuery(ctx, q, query)
			if err != nil {
				return fmt.Errorf("process query %q: %w", query.Title, err)
			}
			if err := q.LinkQueryToReport(
				ctx,
				sqlcgen.LinkQueryToReportParams{ReportID: reportID, QueryID: qID},
			); err != nil {
				return fmt.Errorf("link query %d: %w", qID, err)
			}
		}

		for _, export := range rep.Exports {
			formatID, err := s.getOrCreateExportFormat(ctx, q, export.Format)
			if err != nil {
				return fmt.Errorf("process export format %q: %w", export.Format, err)
			}

			sortOrder, err := json.Marshal(export.Order)
			if err != nil {
				return fmt.Errorf("marshal export order: %w", err)
			}

			if err := q.LinkExportToReport(ctx, sqlcgen.LinkExportToReportParams{
				ReportID:  &reportID,
				FormatID:  &formatID,
				FileName:  export.FileName,
				SortOrder: sortOrder,
			}); err != nil {
				return fmt.Errorf("link export: %w", err)
			}

			if export.Template != nil {
				tmplID, err := s.getOrCreateTemplate(ctx, q, *export.Template)
				if err != nil {
					return fmt.Errorf("process template %q: %w", export.Template.Title, err)
				}
				if err := q.LinkTemplateToReport(
					ctx,
					sqlcgen.LinkTemplateToReportParams{ReportID: reportID, TemplateID: tmplID},
				); err != nil {
					return fmt.Errorf("link template: %w", err)
				}
			}
		}

		for _, cron := range rep.Crons {
			cID, err := s.getOrCreateCron(ctx, q, cron)
			if err != nil {
				return fmt.Errorf("process cron %q: %w", cron.Name, err)
			}
			if err := q.LinkCronToReport(
				ctx,
				sqlcgen.LinkCronToReportParams{ReportID: reportID, CronID: cID},
			); err != nil {
				return fmt.Errorf("link cron %d: %w", cID, err)
			}
		}

		for _, recipient := range rep.Recipients {
			rID, err := s.getOrCreateRecipient(ctx, q, recipient)
			if err != nil {
				return fmt.Errorf("process recipient %q: %w", recipient.Name, err)
			}
			if err := q.LinkRecipientToReport(
				ctx,
				sqlcgen.LinkRecipientToReportParams{ReportID: &reportID, RecipientID: &rID},
			); err != nil {
				return fmt.Errorf("link recipient %d: %w", rID, err)
			}
		}

		return nil
	})

	return int64(reportID), err
}

func (s *ReportStore) getOrCreateEvaluation(
	ctx context.Context,
	q *sqlcgen.Queries,
	eval string,
) (int64, error) {
	id, err := q.FindEvaluationIDByExpr(ctx, eval)
	if err == nil {
		return int64(id), nil
	}
	if translated := translateNoRows(err); !errors.Is(translated, models.ErrNotFound) {
		return 0, fmt.Errorf("find evaluation: %w", err)
	}
	id, err = q.CreateEvaluation(ctx, eval)
	return int64(id), err
}

func (s *ReportStore) createPipeline(
	ctx context.Context,
	q *sqlcgen.Queries,
	pipe *models.Pipeline,
) (int64, error) {
	pipeBytes, err := json.Marshal(pipe)
	if err != nil {
		return 0, fmt.Errorf("marshal pipeline: %w", err)
	}
	id, err := q.CreatePipeline(ctx, pipeBytes)
	return int64(id), err
}

func (s *ReportStore) getOrCreateQuery(
	ctx context.Context,
	q *sqlcgen.Queries,
	card models.Card,
) (int32, error) {
	id, err := q.FindQueryIDByUUIDAndTitle(ctx, sqlcgen.FindQueryIDByUUIDAndTitleParams{
		CardUuid: card.CardUUID, Title: card.Title,
	})
	if err == nil {
		return id, nil
	}
	if translated := translateNoRows(err); !errors.Is(translated, models.ErrNotFound) {
		return 0, fmt.Errorf("find query: %w", err)
	}

	// card.RawParams заполняется только при чтении карточки из БД (тег json:"-"
	// исключает его из JSON-декодирования). Путь создания отчёта
	// (ReportManager.Create -> json.Decoder) заполняет только card.Params,
	// поэтому RawParams здесь всегда nil — раньше это приводило к тому, что
	// реальные параметры карточки молча терялись и записывались как "{}".
	var params []byte

	switch {
	case len(card.RawParams) > 0:
		params = card.RawParams
	case len(card.Params) > 0:
		var err error

		params, err = json.Marshal(card.Params)
		if err != nil {
			return 0, fmt.Errorf("marshal card params: %w", err)
		}
	default:
		params = []byte("{}")
	}

	return q.CreateQuery(ctx, sqlcgen.CreateQueryParams{
		CardUuid: card.CardUUID, Title: card.Title, QType: card.Type, Params: params,
	})
}

func (s *ReportStore) getOrCreateExportFormat(
	ctx context.Context,
	q *sqlcgen.Queries,
	format string,
) (int32, error) {
	id, err := q.FindExportFormatIDByFormat(ctx, &format)
	if err == nil {
		return id, nil
	}
	if translated := translateNoRows(err); !errors.Is(translated, models.ErrNotFound) {
		return 0, fmt.Errorf("find export format: %w", err)
	}
	return q.CreateExportFormat(ctx, &format)
}

func (s *ReportStore) getOrCreateTemplate(
	ctx context.Context,
	q *sqlcgen.Queries,
	tmpl models.Template,
) (int32, error) {
	id, err := q.FindTemplateIDByTitleAndType(ctx, sqlcgen.FindTemplateIDByTitleAndTypeParams{
		Title: &tmpl.Title, Type: tmpl.Type,
	})
	if err == nil {
		return id, nil
	}
	if translated := translateNoRows(err); !errors.Is(translated, models.ErrNotFound) {
		return 0, fmt.Errorf("find template: %w", err)
	}
	return q.CreateTemplate(ctx, sqlcgen.CreateTemplateParams{
		TemplateText: &tmpl.TemplateText, Title: &tmpl.Title, Type: tmpl.Type,
	})
}

func (s *ReportStore) getOrCreateCron(
	ctx context.Context,
	q *sqlcgen.Queries,
	cron models.Cron,
) (int32, error) {
	id, err := q.FindCronIDByNameAndExpr(ctx, sqlcgen.FindCronIDByNameAndExprParams{
		Name: cron.Name, Cron: cron.Cron,
	})
	if err == nil {
		return id, nil
	}
	if translated := translateNoRows(err); !errors.Is(translated, models.ErrNotFound) {
		return 0, fmt.Errorf("find cron: %w", err)
	}
	return q.CreateCron(ctx, sqlcgen.CreateCronParams{
		Cron:        cron.Cron,
		Name:        cron.Name,
		Description: &cron.Description,
		IsActive:    cron.IsActive,
		EventType:   int32(cron.EventType), //nolint:gosec // small enum value from config
	})
}

func (s *ReportStore) getOrCreateRecipient(
	ctx context.Context,
	q *sqlcgen.Queries,
	recipient models.Recipient,
) (int32, error) {
	id, err := q.FindRecipientIDByName(ctx, recipient.Name)
	if err == nil {
		return id, nil
	}
	if translated := translateNoRows(err); !errors.Is(translated, models.ErrNotFound) {
		return 0, fmt.Errorf("find recipient: %w", err)
	}
	return s.createRecipient(ctx, q, recipient)
}

func (s *ReportStore) createRecipient(
	ctx context.Context,
	q *sqlcgen.Queries,
	recipient models.Recipient,
) (int32, error) {
	var chatID, emailID *int32

	if recipient.Chat != nil {
		id, err := s.getOrCreateChat(ctx, q, *recipient.Chat)
		if err != nil {
			return 0, err
		}
		chatID = &id
	}

	if recipient.Email != nil {
		id, err := q.CreateEmailTemplate(ctx, sqlcgen.CreateEmailTemplateParams{
			Dest: recipient.Email.Dest, Copy: recipient.Email.Copy,
			Subject: recipient.Email.Subject, Body: recipient.Email.Body,
		})
		if err != nil {
			return 0, err
		}
		emailID = &id
	}

	var threadID *int32
	if recipient.ThreadID != nil {
		v := int32(*recipient.ThreadID) //nolint:gosec // telegram thread ids fit int32
		threadID = &v
	}

	recipientType := string(recipient.Type)

	return q.CreateRecipient(ctx, sqlcgen.CreateRecipientParams{
		Name: recipient.Name, RemotePath: recipient.RemotePath, ChatID: chatID,
		ThreadID: threadID, EmailID: emailID, Type: &recipientType,
		NeedDeleteAfterEndOfDay: &recipient.NeedDeleteAfterEndOfDay,
	})
}

func (s *ReportStore) getOrCreateChat(
	ctx context.Context,
	q *sqlcgen.Queries,
	chat models.Chat,
) (int32, error) {
	id, err := q.FindChatIDByChatID(ctx, chat.ChatID)
	if err == nil {
		return id, nil
	}
	if translated := translateNoRows(err); !errors.Is(translated, models.ErrNotFound) {
		return 0, err
	}
	return q.CreateChat(ctx, sqlcgen.CreateChatParams{
		ChatID: chat.ChatID, Title: chat.Title, Type: chat.Type,
		Description: chat.Description, IsActive: chat.IsActive, ChType: chat.ChType,
	})
}

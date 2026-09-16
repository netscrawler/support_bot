package store

import (
	"context"
	"encoding/json"
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

func NewReportStore(pool *pgxpool.Pool, log *slog.Logger) *ReportStore {
	return &ReportStore{
		pool: pool,
		q:    sqlcgen.New(pool),
		log:  log.With(slog.Any("module", "store.report")),
	}
}

// NewReportStoreForTest builds a ReportStore around an already-constructed
// *sqlcgen.Queries (e.g. one backed by a pgxmock pool) for unit tests that
// don't need ExecTx/transaction machinery. Production code always uses
// NewReportStore.
func NewReportStoreForTest(q *sqlcgen.Queries) *ReportStore {
	return &ReportStore{q: q, log: slog.Default()}
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
	const pageSize = 5

	if page <= 0 {
		page = 1
	}

	total, err := s.q.CountReportsForLK(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count reports: %w", err)
	}

	offset := int32(page-1) * pageSize

	rows, err := s.q.ListReportsForLK(
		ctx,
		sqlcgen.ListReportsForLKParams{Limit: pageSize, Offset: offset},
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
			return nil, fmt.Errorf("load pipeline: %w", err)
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
		Exports:      mapExportRows(exportRows),
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

func mapExportRows(rows []sqlcgen.ListExportsByReportIDRow) []models.Export {
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
			_ = json.Unmarshal(row.SortOrder, &order)
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

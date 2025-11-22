package pgrepo

import (
	"context"

	gen "support_bot/internal/infra/out/pg/gen"
	"support_bot/internal/models"
)

type Report struct {
	q *gen.Queries
}

func NewQuery(s gen.DBTX) *Report {
	q := gen.New(s)

	return &Report{
		q: q,
	}
}

func (q *Report) GetAll(ctx context.Context) ([]models.Report, error) {
	g, err := q.q.ListAllNotifies(ctx)
	if err != nil {
		return nil, err
	}

	retGroups := make([]models.Report, 0, len(g))

	for _, g := range g {
		if g.RemotePath.String != "" {
			q, err := models.NewReport(
				g.Name,
				g.GroupID.String,
				g.CardUuids,
				g.Cron,
				&g.TemplateText.String,
				g.Title,
				g.GroupTitle.String,
				g.ChatID.Int64,
				&g.RemotePath.String,
				int(g.ThreadID),
				models.TargetFileServerKind,
				g.Active,
				g.Format,
			)
			if err != nil {
				return nil, err
			}

			retGroups = append(retGroups, q)

			continue
		}

		if g.ChatID.Int64 != 0 {
			q, err := models.NewReport(
				g.Name,
				g.GroupID.String,
				g.CardUuids,
				g.Cron,
				&g.TemplateText.String,
				g.Title,
				g.GroupTitle.String,
				g.ChatID.Int64,
				nil,
				int(g.ThreadID),
				models.TargetTelegramChatKind,
				g.Active,
				g.Format,
			)
			if err != nil {
				return nil, err
			}

			retGroups = append(retGroups, q)
		}
	}

	return retGroups, nil
}

func (q *Report) GetAllActive(ctx context.Context) ([]models.Report, error) {
	g, err := q.q.ListAllActiveNotifies(ctx)
	if err != nil {
		return nil, err
	}

	retGroups := make([]models.Report, 0, len(g))

	for _, g := range g {
		if g.RemotePath.String != "" {
			q, err := models.NewReport(
				g.Name,
				g.GroupID.String,
				g.CardUuids,
				g.Cron,
				&g.TemplateText.String,
				g.Title,
				g.GroupTitle.String,
				g.ChatID.Int64,
				&g.RemotePath.String,
				int(g.ThreadID),
				models.TargetFileServerKind,
				g.Active,
				g.Format,
			)
			if err != nil {
				return nil, err
			}

			retGroups = append(retGroups, q)

			continue
		}

		if g.ChatID.Int64 != 0 {
			q, err := models.NewReport(
				g.Name,
				g.GroupID.String,
				g.CardUuids,
				g.Cron,
				&g.TemplateText.String,
				g.Title,
				g.GroupTitle.String,
				g.ChatID.Int64,
				nil,
				int(g.ThreadID),
				models.TargetTelegramChatKind,
				g.Active,
				g.Format,
			)
			if err != nil {
				return nil, err
			}

			retGroups = append(retGroups, q)
		}
	}

	return retGroups, nil
}

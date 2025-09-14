package pgrepo

import (
	"context"
	"support_bot/internal/models"

	gen "support_bot/internal/infra/out/pg/gen"
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
		q, err := models.NewReport(
			g.Name,
			g.GroupID.String,
			g.CardUuid.String,
			g.Cron,
			&g.TemplateText.String,
			g.Title,
			g.GroupTitle.String,
			g.ChatID.Int64,
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

	return retGroups, nil
}

func (q *Report) GetAllActive(ctx context.Context) ([]models.Report, error) {
	g, err := q.q.ListAllActiveNotifies(ctx)
	if err != nil {
		return nil, err
	}

	retGroups := make([]models.Report, 0, len(g))

	for _, g := range g {
		q, err := models.NewReport(
			g.Name,
			g.GroupID.String,
			g.CardUuid.String,
			g.Cron,
			&g.TemplateText.String,
			g.Title,
			g.GroupTitle.String,
			g.ChatID.Int64,
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

	return retGroups, nil
}

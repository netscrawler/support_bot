package pgrepo

import (
	"context"
	"support_bot/internal/models"

	gen "support_bot/internal/infra/out/pg/gen"
)

type Notify struct {
	q *gen.Queries
}

func NewQuery(s gen.DBTX) *Notify {
	q := gen.New(s)

	return &Notify{
		q: q,
	}
}

func (q *Notify) GetAll(ctx context.Context) ([]models.Notify, error) {
	g, err := q.q.ListAllNotifies(ctx)
	if err != nil {
		return nil, err
	}

	retGroups := make([]models.Notify, 0, len(g))

	for _, g := range g {
		q, err := models.NewNotify(
			g.Name,
			g.GroupID.String,
			g.CardUuid,
			g.Cron,
			&g.TemplateText.String,
			g.Title.String,
			g.GroupTitle.String,
			g.ChatID,
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

func (q *Notify) GetAllActive(ctx context.Context) ([]models.Notify, error) {
	g, err := q.q.ListAllActiveNotifies(ctx)
	if err != nil {
		return nil, err
	}

	retGroups := make([]models.Notify, 0, len(g))

	for _, g := range g {
		q, err := models.NewNotify(
			g.Name,
			g.GroupID.String,
			g.CardUuid,
			g.Cron,
			&g.TemplateText.String,
			g.Title.String,
			g.GroupTitle.String,
			g.ChatID,
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

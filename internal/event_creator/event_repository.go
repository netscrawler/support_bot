package eventcreator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type EventRepository struct {
	db *sqlx.DB

	log *slog.Logger
}

func NewRepository(db *sqlx.DB, log *slog.Logger) *EventRepository {
	l := log.With("module", "event_repository")

	return &EventRepository{
		db:  db,
		log: l,
	}
}

func (er *EventRepository) Load(ctx context.Context) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("event repository load : %w", err)
	}

	const query string = `select c.name as cron_name, r.name as report_name
from report_crons rc
join crons c on c.id = rc.cron_id
join reports r on r.id = rc.report_id
;

`

	var events []Event

	err := er.db.SelectContext(ctx, &events, query)
	if err != nil {
		er.log.ErrorContext(ctx, "error loading events", slog.Any("error", err))

		return nil, err
	}

	er.log.InfoContext(ctx, "loaded events", slog.Any("count", len(events)))

	return events, nil
}

func (er *EventRepository) LoadByName(ctx context.Context, name string) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("event repository load by name : %w", err)
	}

	const query string = `select c.name as cron_name, r.name as report_name
from report_crons rc
join crons c on c.id = rc.cron_id
join reports r on r.id = rc.report_id
where c.name = $1
;

`

	var events []Event

	err := er.db.SelectContext(ctx, &events, query, name)
	if err != nil {
		er.log.ErrorContext(ctx, "error loading events by name", slog.Any("error", err))

		return nil, err
	}

	er.log.InfoContext(ctx, "loaded events", slog.Any("count", len(events)))

	return events, nil
}

package sheduler

import (
	"context"
	"log/slog"

	models "support_bot/internal/models/report"

	"github.com/jmoiron/sqlx"
)

type SheduleRepo struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewSheduleRepo(db *sqlx.DB, log *slog.Logger) *SheduleRepo {
	l := log.With(slog.Any("module", "shedule_repository"))

	return &SheduleRepo{
		db:  db,
		log: l,
	}
}

func (s *SheduleRepo) Load(ctx context.Context) ([]models.SheduleUnit, error) {
	const query string = `select cron, name from crons where is_active = true`

	s.log.DebugContext(ctx, "start loading shedules")

	var units []models.SheduleUnit

	err := s.db.SelectContext(ctx, &units, query)
	if err != nil {
		s.log.ErrorContext(ctx, "error loading shedules", slog.Any("error", err))

		return nil, err
	}

	return units, nil
}

package sheduler

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
	models "support_bot/internal/models/report"
)

type SheduleLoader interface {
	Load(ctx context.Context) ([]models.SheduleUnit, error)
}

type Sheduler struct {
	cron   *cron.Cron
	log    *slog.Logger
	loader SheduleLoader

	EventChan chan string
}

func NewSheduler(shLoader SheduleLoader, log *slog.Logger, events chan string) *Sheduler {
	l := log.With(slog.Any("module", "sheduler"))

	return &Sheduler{
		cron:      cron.New(),
		log:       l,
		loader:    shLoader,
		EventChan: events,
	}
}

func (s *Sheduler) Start(ctx context.Context) error {
	s.cron.Stop()
	s.cron = cron.New()
	s.log.InfoContext(ctx, "Starting")

	units, err := s.loader.Load(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "Error while loading shedule", slog.Any("error", err))

		return err
	}

	for _, u := range units {
		entry, err := s.cron.AddFunc(u.Crontab, func() {
			go func() {
				s.log.Debug("cron job executed", slog.Any("job_name", u.Name))

				s.EventChan <- u.Name
			}()
		})
		if err != nil {
			s.log.ErrorContext(ctx, "Error start job", slog.Any("job", u), slog.Any("error", err))

			continue
		}

		s.log.InfoContext(ctx, "Started job", slog.Any("job", u), slog.Any("entry", entry))
	}

	s.log.InfoContext(ctx, "Sheduler started")

	s.cron.Start()

	return nil
}

func (s *Sheduler) Stop() {
	s.cron.Stop()
}

package sheduler

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
)

type SheduleLoader interface {
	Load(ctx context.Context) ([]SheduleUnit, error)
}

type SheduleUnit struct {
	Crontab string
	Name    string
}

type Sheduler struct {
	cron   *cron.Cron
	log    *slog.Logger
	loader SheduleLoader

	EventChan chan string
}

func NewSheduler(shLoader SheduleLoader, log *slog.Logger) *Sheduler {
	l := log.WithGroup("sheduler")
	events := make(chan string, 100)
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
	s.log.InfoContext(ctx, "Starting sheduler")
	units, err := s.loader.Load(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "Error while loading shedule", slog.Any("error", err))
		return err
	}

	for _, u := range units {
		entry, err := s.cron.AddFunc(u.Crontab, func() {
			s.EventChan <- u.Name
		})
		if err != nil {
			s.log.ErrorContext(ctx, "Error start job", slog.Any("job", u), slog.Any("error", err))

			continue
		}
		s.log.InfoContext(ctx, "Started job", slog.Any("job", u), slog.Any("entry", entry))
	}
	s.log.InfoContext(ctx, "Sheduler started")

	return nil
}

func (s *Sheduler) Stop() {
	s.cron.Stop()
}

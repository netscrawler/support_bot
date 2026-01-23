package sheduler

import (
	"context"
	"log/slog"

	models "support_bot/internal/models/report"

	"github.com/robfig/cron/v3"
)

type SheduleLoader interface {
	Load(ctx context.Context) ([]models.SheduleUnit, error)
}

type Sheduler struct {
	cron   *cron.Cron
	log    *slog.Logger
	loader SheduleLoader

	EventChan chan string

	api chan SheduleAPIEvent
}

func NewSheduler(
	shLoader SheduleLoader,
	log *slog.Logger,
	events chan string,
	apiChan chan SheduleAPIEvent,
) *Sheduler {
	l := log.With(slog.Any("module", "sheduler"))

	return &Sheduler{
		cron:      cron.New(),
		log:       l,
		loader:    shLoader,
		EventChan: events,
		api:       apiChan,
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
	s.StartMonitor(ctx)

	return nil
}

func (s *Sheduler) Stop() {
	s.log.Info("stoping sheduler")
	s.cron.Stop()
}

func (s *Sheduler) StartMonitor(ctx context.Context) {
	s.log.DebugContext(ctx, "starting event monitor")
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-s.api:
				switch ev {
				case EventStart:
					s.log.DebugContext(ctx, "received start event")
					s.Stop()
					s.Start(ctx)
				case EventStop:
					s.log.DebugContext(ctx, "received stop event")
					s.Stop()
				default:
					s.log.DebugContext(ctx, "received unknown event", slog.Any("event", ev))
				}
			}
		}
	}()
}

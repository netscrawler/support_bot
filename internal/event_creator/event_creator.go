package eventcreator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"support_bot/internal/models"
)

type event struct {
	Name     string `db:"report_name"`
	CronName string `db:"cron_name"`
}

type EventProvider interface {
	Load(ctx context.Context) ([]event, error)
	LoadByName(ctx context.Context, name string) ([]event, error)
}

type EventCreator struct {
	mu    sync.RWMutex
	cache map[string][]models.Event
	InC   chan models.Event
	OutC  chan models.Event

	log *slog.Logger
	ep  EventProvider
}

func New(
	input chan models.Event,
	out chan models.Event,
	log *slog.Logger,
	ep EventProvider,
) *EventCreator {
	l := log.With(slog.Any("module", "event_creator"))

	return &EventCreator{
		cache: make(map[string][]models.Event),
		InC:   input,
		OutC:  out,
		log:   l,
		ep:    ep,
		mu:    sync.RWMutex{},
	}
}

func (e *EventCreator) Start(ctx context.Context) error {
	e.cleaner(ctx)

	err := e.heat(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				e.log.InfoContext(ctx, "context cancelled")

				return
			case ev, ok := <-e.InC:
				if !ok {
					e.log.ErrorContext(ctx, "input chan closed")

					return
				}

				gCtx, cancel := context.WithTimeout(ctx, 15*time.Second)

				switch ev.Type {
				case models.EventTypeGenReport:
					e.createGenReportEvent(gCtx, ev)
				case models.EventTypeDeleteSentReport:
					e.createDeleteSentReportEvent(gCtx, ev)

				default:
					e.createGenReportEvent(gCtx, ev)
				}

				cancel()
			}
		}
	}()

	return nil
}

func (e *EventCreator) reLoad() {
	e.mu.Lock()
	clear(e.cache)
	e.mu.Unlock()
}

func (e *EventCreator) createGenReportEvent(ctx context.Context, ev models.Event) {
	events, err := e.getByCronName(ctx, ev.Name)
	if err != nil {
		e.log.ErrorContext(ctx, "error load events", slog.Any("error", err))

		return
	}

	for _, en := range events {
		select {
		case <-ctx.Done():
			e.log.InfoContext(ctx, "context cancelled")

			return
		case e.OutC <- en:
			e.log.DebugContext(ctx, "sending report event", slog.Any("event", ev))
		}
	}
}

func (e *EventCreator) createDeleteSentReportEvent(ctx context.Context, ev models.Event) {
	select {
	case <-ctx.Done():
		e.log.InfoContext(ctx, "context cancelled")

		return
	case e.OutC <- ev:
		e.log.DebugContext(ctx, "sending deleting event", slog.Any("event", ev))
	}
}

func (e *EventCreator) getByCronName(ctx context.Context, name string) ([]models.Event, error) {
	e.mu.RLock()

	if ev, ok := e.cache[name]; ok {
		e.mu.RUnlock()

		return ev, nil
	}

	e.mu.RUnlock()

	events, err := e.ep.LoadByName(ctx, name)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	var names []models.Event

	for _, ev := range events {
		names = append(names, models.Event{
			Name: ev.Name,
			Type: models.EventTypeGenReport,
		})

		events, ok := e.cache[ev.CronName]
		if !ok {
			e.cache[ev.CronName] = []models.Event{{Name: name, Type: models.EventTypeGenReport}}

			continue
		}

		e.cache[ev.CronName] = append(
			events,
			models.Event{Name: name, Type: models.EventTypeGenReport},
		)
	}

	return names, nil
}

func (e *EventCreator) cleaner(ctx context.Context) {
	tick := time.NewTicker(5 * time.Minute)

	go func() {
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				e.reLoad()
				e.log.DebugContext(ctx, "cache cleaned")
			}
		}
	}()
}

func (e *EventCreator) heat(ctx context.Context) error {
	e.log.DebugContext(ctx, "start loading events")

	events, err := e.ep.Load(ctx)
	if err != nil {
		e.log.ErrorContext(ctx, "error loading events", slog.Any("error", err))

		return fmt.Errorf("errorl loading events: (%w)", err)
	}

	e.log.DebugContext(
		ctx,
		"event loaded successfully, start save to cache",
		slog.Any("events_count", len(events)),
	)

	e.mu.Lock()

	for _, ev := range events {
		events, ok := e.cache[ev.CronName]
		if !ok {
			e.cache[ev.CronName] = []models.Event{{Name: ev.Name, Type: models.EventTypeGenReport}}

			continue
		}

		e.cache[ev.CronName] = append(
			events,
			models.Event{Name: ev.Name, Type: models.EventTypeGenReport},
		)
	}

	e.mu.Unlock()
	e.log.DebugContext(ctx, "successfully add events to cache")

	return nil
}

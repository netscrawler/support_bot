package eventcreator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Event struct {
	Name     string `db:"report_name"`
	CronName string `db:"cron_name"`
}

type EventProvider interface {
	Load(ctx context.Context) ([]Event, error)
	LoadByName(ctx context.Context, name string) ([]Event, error)
}

type EventCreator struct {
	mu    sync.RWMutex
	cache map[string][]string
	InC   chan string
	OutC  chan string

	log *slog.Logger
	ep  EventProvider
}

func New(
	input chan string,
	out chan string,
	log *slog.Logger,
	ep EventProvider,
) *EventCreator {
	l := log.With(slog.Any("module", "event_creator"))

	return &EventCreator{
		cache: make(map[string][]string),
		InC:   input,
		OutC:  out,
		log:   l,
		ep:    ep,
		mu:    sync.RWMutex{},
	}
}

func (e *EventCreator) Start(ctx context.Context) error {
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

				events, err := e.getByCronName(gCtx, ev)
				if err != nil {
					cancel()
					e.log.ErrorContext(ctx, "error load events", slog.Any("error", err))

					continue
				}

				cancel()

				for _, en := range events {
					select {
					case <-ctx.Done():
						e.log.InfoContext(ctx, "context cancelled")

						return
					case e.OutC <- en:
					}
				}
			}
		}
	}()

	return nil
}

func (e *EventCreator) ReLoad() {
	e.mu.Lock()
	clear(e.cache)
	e.mu.Unlock()
}

func (e *EventCreator) getByCronName(ctx context.Context, name string) ([]string, error) {
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

	for _, ev := range events {
		events, ok := e.cache[ev.CronName]
		if !ok {
			e.cache[ev.CronName] = []string{ev.Name}

			continue
		}

		e.cache[ev.CronName] = append(events, ev.Name)
	}

	names, ok := e.cache[name]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	return names, nil
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
			e.cache[ev.CronName] = []string{ev.Name}

			continue
		}

		e.cache[ev.CronName] = append(events, ev.Name)
	}

	e.mu.Unlock()
	e.log.DebugContext(ctx, "successfully add events to cache")

	return nil
}

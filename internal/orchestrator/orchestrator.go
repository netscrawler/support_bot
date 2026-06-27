package orchestrator

import (
	"context"
	"log/slog"
	"sync"
	"time"

	models "support_bot/internal/models/report"
)

type ReportLoader interface {
	Load(ctx context.Context) ([]models.Report, error)
	LoadByEvent(ctx context.Context, event string, active bool) (*models.Report, error)
}

type Orchestrator struct {
	EventC        chan models.Event
	SpecialEventC chan models.SpecialEventForLK

	ReportC chan models.Report
	DeleteC chan models.Event

	rL ReportLoader

	mu    sync.RWMutex
	cache map[string][]models.Report

	log *slog.Logger
}

func New(
	evC chan models.Event,
	specialEventC chan models.SpecialEventForLK,
	reportC chan models.Report,
	delC chan models.Event,
	rl ReportLoader,
	log *slog.Logger,
) *Orchestrator {
	l := log.With(slog.Any("module", "orchestrator"))
	cache := make(map[string][]models.Report)

	return &Orchestrator{
		EventC:        evC,
		SpecialEventC: specialEventC,
		ReportC:       reportC,
		DeleteC:       delC,
		rL:            rl,
		cache:         cache,
		log:           l,
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.log.InfoContext(ctx, "starting...")

	go o.run(ctx)

	o.cleaner(ctx)
}

func (o *Orchestrator) ReLoad() {
	o.mu.Lock()
	clear(o.cache)
	o.mu.Unlock()
}

func (o *Orchestrator) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			o.log.InfoContext(ctx, "context cancelled. stopping")

			return
		case event, ok := <-o.EventC:
			if !ok {
				o.log.WarnContext(ctx, "event chan closed")

				return
			}

			switch event.Type {
			case models.EventTypeDeleteSentReport:
				o.processDelReportEvent(ctx, event.Name)

			default:
				o.processGenReportEvent(ctx, event.Name)

			}
		case event, ok := <-o.SpecialEventC:
			if !ok {
				o.log.WarnContext(ctx, "event chan closed")

				return
			}

			switch event.Event.Type {
			case models.EventTypeGenReportForTG:
				o.processGenReportSpecialEvent(ctx, event)

			default:

			}

		}
	}
}

func (o *Orchestrator) processGenReportEvent(ctx context.Context, event string) {
	reports, err := o.getReportByEvent(ctx, event, true)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading report", slog.Any("error", err))

		return
	}

	for _, report := range reports {
		select {
		case <-ctx.Done():
			o.log.InfoContext(ctx, "context cancelled. stopping")

			return
		case o.ReportC <- report:
			o.log.DebugContext(
				ctx,
				"sending report to generator",
				slog.Any("report", report.Name),
			)
		}
	}
}

func (o *Orchestrator) processGenReportSpecialEvent(ctx context.Context, event models.SpecialEventForLK) {
	reports, err := o.getReportByEvent(ctx, event.Event.Name, false)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading report", slog.Any("error", err))

		return
	}

	for _, report := range reports {
		report.Recipients = []models.Recipient{event.Recipient}
		select {
		case <-ctx.Done():
			o.log.InfoContext(ctx, "context cancelled. stopping")

			return
		case o.ReportC <- report:
			o.log.DebugContext(
				ctx,
				"sending report to generator",
				slog.Any("report", report.Name),
			)
		}
	}
}

func (o *Orchestrator) processDelReportEvent(ctx context.Context, event string) {
	select {
	case <-ctx.Done():
		o.log.InfoContext(ctx, "context cancelled. stopping")
		return
	case o.DeleteC <- models.Event{Name: event, Type: models.EventTypeDeleteSentReport}:
		o.log.InfoContext(ctx, "sending delete event to deleter")
	}
}

func (o *Orchestrator) cleaner(ctx context.Context) {
	tick := time.NewTicker(5 * time.Minute)

	go func() {
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				o.ReLoad()
				o.log.DebugContext(ctx, "cache cleaned")
			}
		}
	}()
}

func (o *Orchestrator) getReportByEvent(
	ctx context.Context,
	event string,
	active bool,
) ([]models.Report, error) {
	l := o.log.With(slog.Any("event", event))
	l.DebugContext(ctx, "getting report by event")

	o.mu.RLock()

	r, ok := o.cache[event]
	if ok {
		l.DebugContext(ctx, "find report in cache")
		o.mu.RUnlock()

		return r, nil
	}

	o.mu.RUnlock()

	l.DebugContext(ctx, "cache miss, loading report")

	reports, err := o.rL.LoadByEvent(ctx, event, active)
	if err != nil {
		l.ErrorContext(ctx, "error while loading report", slog.Any("error", err))

		return nil, err
	}

	if !active {
		return []models.Report{*reports}, nil
	}

	l.DebugContext(ctx, "reports loaded", slog.Any("reports_count", 1))

	o.mu.Lock()
	defer o.mu.Unlock()

	if rp, ok := o.cache[event]; ok {
		o.cache[event] = append(rp, *reports)
	} else {
		o.cache[event] = append([]models.Report{}, *reports)
	}

	return []models.Report{*reports}, nil
}

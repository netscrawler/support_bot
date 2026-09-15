package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"support_bot/internal/generator"
	"support_bot/internal/models"
	"sync"
	"time"
)

type ReportLoader interface {
	Load(ctx context.Context) ([]models.Report, error)
	LoadByEvent(ctx context.Context, event string, active bool) (*models.Report, error)
}

// SentMsgSaver records a delivered message so the deleter can later clean it
// up (e.g. end-of-day Telegram messages).
type SentMsgSaver interface {
	SaveTgMsg(ctx context.Context, reportName string, msgs []models.SentMessage) error
}

type Orchestrator struct {
	EventC        chan models.Event
	SpecialEventC chan models.SpecialEventForLK

	ReportC chan generator.Job
	DeleteC chan models.Event

	rL ReportLoader

	snd         models.SenderProvider
	sentMsgRepo SentMsgSaver

	mu    sync.RWMutex
	cache map[string][]models.Report

	log *slog.Logger
}

func New(
	evC chan models.Event,
	specialEventC chan models.SpecialEventForLK,
	reportC chan generator.Job,
	delC chan models.Event,
	rl ReportLoader,
	snd models.SenderProvider,
	sentMsgRepo SentMsgSaver,
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
		snd:           snd,
		sentMsgRepo:   sentMsgRepo,
		cache:         cache,
		log:           l,
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.log.InfoContext(ctx, "starting...")

	go o.run(ctx)

	o.cleaner(ctx)
}

func (o *Orchestrator) reLoad() {
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
			case models.EventTypeGenReport:
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
		result := make(chan generator.JobResult, 1)

		select {
		case <-ctx.Done():
			o.log.InfoContext(ctx, "context cancelled. stopping")

			return
		case o.ReportC <- generator.Job{Report: report, Result: result}:
			o.log.DebugContext(
				ctx,
				"sending report to generator",
				slog.Any("report", report.Name),
			)
		}

		go o.awaitAndDeliver(ctx, report, result)
	}
}

func (o *Orchestrator) processGenReportSpecialEvent(
	ctx context.Context,
	event models.SpecialEventForLK,
) {
	reports, err := o.getReportByEvent(ctx, event.Event.Name, false)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading report", slog.Any("error", err))

		return
	}

	for _, report := range reports {
		if event.Event.Type == models.EventTypeGenReportForTG {
			report.Recipients = []models.Recipient{event.Recipient}
		}

		result := make(chan generator.JobResult, 1)

		select {
		case <-ctx.Done():
			o.log.InfoContext(ctx, "context cancelled. stopping")

			return
		case o.ReportC <- generator.Job{Report: report, Result: result}:
			o.log.DebugContext(
				ctx,
				"sending report to generator",
				slog.Any("report", report.Name),
			)
		}

		go o.awaitAndDeliver(ctx, report, result)
	}
}

// awaitAndDeliver waits for the generator's result for report and, on a
// positive evaluation, delivers it to report's recipients. Generation
// already happened in the shared worker pool; this just picks up where it
// left off, so it runs in its own goroutine and never blocks the event loop.
func (o *Orchestrator) awaitAndDeliver(
	ctx context.Context,
	report models.Report,
	result <-chan generator.JobResult,
) {
	select {
	case <-ctx.Done():
		return
	case r := <-result:
		if r.Err != nil {
			o.log.ErrorContext(ctx, "error create report", slog.Any("error", r.Err))

			return
		}

		if !r.Approve {
			return
		}

		if err := o.deliver(ctx, report, r.Dataset, r.Data); err != nil {
			o.log.ErrorContext(ctx, "error delivering report", slog.Any("error", err))
		}
	}
}

// deliver sends a generated report to its configured recipients and records
// the resulting message state.
func (o *Orchestrator) deliver(
	ctx context.Context,
	report models.Report,
	data models.Dataset,
	res []models.Data,
) error {
	l := o.log

	if len(report.Recipients) == 0 {
		l.ErrorContext(ctx, "empty targets list")

		return fmt.Errorf("empty targets list")
	}

	// Достаем "_meta" лист из данных, для использования в шаблоне email
	addMeta := make(map[string]any)
	meta, ok := data["_meta"]
	if ok {
		for _, d := range meta {
			maps.Insert(addMeta, maps.All(d))
		}
	}
	l.InfoContext(ctx, "meta", slog.Any("meta", addMeta), slog.Any("_meta", data["_meta"]))
	msg := models.NewMessage(report.Name, res, addMeta, report.Recipients...)

	resMsg, err := msg.Send(ctx, o.snd)
	if err != nil {
		l.ErrorContext(ctx, "error while send message", slog.Any("error", err))
	}

	if len(resMsg) == 0 {
		l.InfoContext(ctx, "report generated")

		return nil
	}

	l.InfoContext(
		ctx,
		"saving message to database",
		slog.Any("report", report.Name),
		slog.Any("message", resMsg),
	)

	err = o.sentMsgRepo.SaveTgMsg(ctx, msg.ReportName, resMsg)
	if err != nil {
		l.WarnContext(ctx, "result msg save failed", slog.Any("error", err))
	}

	return nil
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
				o.reLoad()
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

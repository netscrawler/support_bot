package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"support_bot/internal/generator"
	"support_bot/internal/models"
	"support_bot/internal/pkg/logger"
)

// ponytail: fixed-size semaphore, not wired to the generator's actual worker
// count (hardcoded 4 in app.go, itself slated to move under fx) — sync the
// two or thread this through New() if throughput needs tuning.
const maxInFlightGenerations = 8

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

	DeleteC chan models.Event

	rL  ReportLoader
	gen *generator.Generator

	snd         models.SenderProvider
	sentMsgRepo SentMsgSaver

	genSem chan struct{}

	log *slog.Logger
}

func New(
	evC chan models.Event,
	specialEventC chan models.SpecialEventForLK,
	delC chan models.Event,
	rl ReportLoader,
	gen *generator.Generator,
	snd models.SenderProvider,
	sentMsgRepo SentMsgSaver,
	log *slog.Logger,
) *Orchestrator {
	l := log.With(slog.Any("module", "orchestrator"))

	return &Orchestrator{
		EventC:        evC,
		SpecialEventC: specialEventC,
		DeleteC:       delC,
		rL:            rl,
		gen:           gen,
		snd:           snd,
		sentMsgRepo:   sentMsgRepo,
		genSem:        make(chan struct{}, maxInFlightGenerations),
		log:           l,
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.log.InfoContext(ctx, "starting...")

	go o.run(ctx)
}

// Generate implements service.ReportGenerator's single-file Generate
// signature (internal/service/report_generator.go), routing an on-demand
// request through the exact same generate step as the event-driven path —
// the orchestrator doesn't distinguish "send it" from "return it", only
// what happens with the result differs. On-demand reports are expected to
// declare exactly one export; if more are configured, the first is
// returned — a known simplification. A negative evaluation result, or no
// exports, is surfaced as models.ErrNotFound, matching the "not found"
// semantics the HTTP layer expects.
func (o *Orchestrator) Generate(ctx context.Context, report models.Report) (models.Data, error) {
	_, data, approve, err := o.generate(ctx, report)
	if err != nil {
		return models.Data{}, err
	}

	if !approve || len(data) == 0 {
		return models.Data{}, models.ErrNotFound
	}

	return data[0], nil
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
				o.processGenReportEvent(ctx, event.Name, true, nil)

			}
		case event, ok := <-o.SpecialEventC:
			if !ok {
				o.log.WarnContext(ctx, "event chan closed")

				return
			}

			switch event.Event.Type {
			case models.EventTypeGenReportForTG:
				o.processGenReportEvent(ctx, event.Event.Name, false, &event.Recipient)
			case models.EventTypeGenReport:
				o.processGenReportEvent(ctx, event.Event.Name, false, nil)
			default:
			}
		}
	}
}

// processGenReportEvent loads the report(s) registered for event and
// delivers each one — the caller doesn't distinguish which channel or event
// type triggered this beyond the two things that actually vary: whether the
// event only targets active reports, and an optional recipient override for
// a special one-off (e.g. "generate for this LK chat right now").
func (o *Orchestrator) processGenReportEvent(
	ctx context.Context,
	event string,
	active bool,
	recipientOverride *models.Recipient,
) {
	reports, err := o.getReportByEvent(ctx, event, active)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading report", slog.Any("error", err))

		return
	}

	for _, report := range reports {
		if recipientOverride != nil {
			report.Recipients = []models.Recipient{*recipientOverride}
		}

		o.log.DebugContext(ctx, "sending report to generator", slog.Any("report", report.Name))

		select {
		case o.genSem <- struct{}{}:
			go func(report models.Report) {
				defer func() { <-o.genSem }()

				o.generateAndDeliver(ctx, report)
			}(report)
		case <-ctx.Done():
			return
		}
	}
}

// generate runs report through the generator's shared worker pool under a
// log context. The per-report timeout budget is the worker pool's own
// concern (internal/generator/generator.go), not duplicated here. It stops
// short of doing anything with the result — handing that off, either by
// delivering it (generateAndDeliver) or returning it to an on-demand caller
// (Generate), is up to the callers below.
func (o *Orchestrator) generate(
	ctx context.Context,
	report models.Report,
) (models.Dataset, []models.Data, bool, error) {
	ctx = logger.AppendCtx(ctx, slog.Any("report_name", report.Name))

	return o.gen.Generate(ctx, report)
}

// generateAndDeliver runs report through the generator's shared worker pool
// and, on a positive evaluation, delivers it to report's recipients. It
// runs in its own goroutine so a busy pool never blocks the event loop.
func (o *Orchestrator) generateAndDeliver(ctx context.Context, report models.Report) {
	dataset, data, approve, err := o.generate(ctx, report)
	if err != nil {
		o.log.ErrorContext(ctx, "error create report", slog.Any("error", err))

		return
	}

	if !approve {
		return
	}

	if err := o.deliver(ctx, report, dataset, data); err != nil {
		o.log.ErrorContext(ctx, "error delivering report", slog.Any("error", err))
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

func (o *Orchestrator) getReportByEvent(
	ctx context.Context,
	event string,
	active bool,
) ([]models.Report, error) {
	l := o.log.With(slog.Any("event", event))
	l.DebugContext(ctx, "getting report by event")

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

	return []models.Report{*reports}, nil
}

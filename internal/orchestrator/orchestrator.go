package orchestrator

import (
	"context"
	"log/slog"

	models "support_bot/internal/models/report"

	lru "github.com/hashicorp/golang-lru/v2"
)

type ReportLoader interface {
	Load(ctx context.Context) ([]models.Report, error)
	LoadByEvent(ctx context.Context, event string) (*models.Report, error)
}

type Orchestrator struct {
	EventC chan string

	ReportC chan models.Report

	rL ReportLoader

	cache *lru.Cache[string, []models.Report]

	log *slog.Logger
}

func New(
	evC chan string,
	reportC chan models.Report,
	rl ReportLoader,
	log *slog.Logger,
) *Orchestrator {
	l := log.With(slog.Any("module", "orchestrator"))
	cache, _ := lru.New[string, []models.Report](5)

	return &Orchestrator{
		EventC:  evC,
		ReportC: reportC,
		rL:      rl,
		cache:   cache,
		log:     l,
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.log.InfoContext(ctx, "starting...")

	go o.run(ctx)
}

func (o *Orchestrator) ReLoad() {
	o.cache.Purge()
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

			reports, err := o.getReportByEvent(ctx, event)
			if err != nil {
				o.log.ErrorContext(ctx, "error loading report", slog.Any("error", err))

				continue
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
	}
}

func (o *Orchestrator) getReportByEvent(
	ctx context.Context,
	event string,
) ([]models.Report, error) {
	l := o.log.With(slog.Any("event", event))
	l.DebugContext(ctx, "getting report by event")

	r, ok := o.cache.Get(event)
	if ok {
		l.DebugContext(ctx, "find report in cache")

		return r, nil
	}

	l.DebugContext(ctx, "cache miss, loading report")

	reports, err := o.rL.LoadByEvent(ctx, event)
	if err != nil {
		l.ErrorContext(ctx, "error while loading report", slog.Any("error", err))

		return nil, err
	}

	l.DebugContext(ctx, "reports loaded", slog.Any("reports_count", len(r)))

	if rp, ok := o.cache.Peek(event); ok {
		o.cache.Add(event, append(rp, *reports))
	} else {
		o.cache.Add(event, append([]models.Report{}, *reports))
	}

	return []models.Report{*reports}, nil
}

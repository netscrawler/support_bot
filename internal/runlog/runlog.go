// Package runlog пишет журнал запусков отчётов и уведомляет о смене состояния сбой/норма.
package runlog

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"support_bot/internal/metrics"
	"support_bot/internal/models"
	"time"
)

const (
	maxErrRunes     = 500
	cleanupInterval = 24 * time.Hour
	defaultRetain   = 720 * time.Hour
)

type Store interface {
	Insert(ctx context.Context, run models.Run) error
	LastStatus(ctx context.Context, reportName string) (models.RunStatus, bool, error)
	LastSuccessHash(ctx context.Context, reportName string) (string, bool, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type Recorder struct {
	store   Store
	alerter Alerter
	cfg     Config
	log     *slog.Logger
}

func New(store Store, alerter Alerter, cfg Config, log *slog.Logger) *Recorder {
	if cfg.Retention <= 0 {
		cfg.Retention = defaultRetain
	}
	return &Recorder{store: store, alerter: alerter, cfg: cfg, log: log}
}

func bad(s models.RunStatus) bool {
	return s == models.RunStatusFailed || s == models.RunStatusPartial
}

// Record пишет запуск и для cron/manual уведомляет о переходе сбой <-> норма.
// test и lk в состоянии уведомлений не участвуют.
func (r *Recorder) Record(ctx context.Context, run models.Run) {
	track := run.Kind == models.RunKindCron || run.Kind == models.RunKindManual

	var prev models.RunStatus
	var hasPrev, prevErr bool
	if track {
		var err error
		prev, hasPrev, err = r.store.LastStatus(ctx, run.ReportName)
		if err != nil {
			prevErr = true
			r.log.ErrorContext(
				ctx,
				"runlog: чтение прошлого статуса",
				"report",
				run.ReportName,
				"error",
				err,
			)
		}
	}

	metrics.ReportRuns.WithLabelValues(string(run.Kind), string(run.Status)).Inc()
	metrics.ReportRunDuration.WithLabelValues(string(run.Kind)).
		Observe(run.FinishedAt.Sub(run.StartedAt).Seconds())

	if err := r.store.Insert(ctx, run); err != nil {
		r.log.ErrorContext(ctx, "runlog: запись запуска", "report", run.ReportName, "error", err)
	}
	if !track {
		return
	}

	prevBad := hasPrev && bad(prev)
	var text string
	switch {
	case bad(run.Status) && !prevBad:
		text = fmt.Sprintf(
			"⚠️ Отчёт <b>%s</b> (%s): %s\n<pre>%s</pre>",
			html.EscapeString(run.ReportName),
			run.Kind,
			run.Status,
			html.EscapeString(truncate(run.Error)),
		)
	case !bad(run.Status) && prevBad && !prevErr:
		text = fmt.Sprintf(
			"✅ Отчёт <b>%s</b> снова работает (%s)",
			html.EscapeString(run.ReportName),
			run.Kind,
		)
	default:
		return
	}
	if err := r.alerter.Alert(ctx, text); err != nil {
		r.log.ErrorContext(
			ctx,
			"runlog: отправка уведомления",
			"report",
			run.ReportName,
			"error",
			err,
		)
	}
}

// LastSuccessHash — хэш данных последнего успешного запуска. Ошибка store → ("", false):
// лучше дубль, чем потеря отправки.
func (r *Recorder) LastSuccessHash(ctx context.Context, report string) (string, bool) {
	h, ok, err := r.store.LastSuccessHash(ctx, report)
	if err != nil {
		r.log.WarnContext(
			ctx,
			"runlog: чтение хэша прошлого запуска",
			"report",
			report,
			"error",
			err,
		)

		return "", false
	}

	return h, ok
}

// truncate режет по рунам (до экранирования), чтобы не порвать UTF-8 и HTML-сущности.
func truncate(s string) string {
	rs := []rune(s)
	if len(rs) <= maxErrRunes {
		return s
	}
	return string(rs[:maxErrRunes]) + "…"
}

// Start раз в сутки удаляет записи старше retention; выходит по ctx.Done().
func (r *Recorder) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(cleanupInterval)
		defer t.Stop()
		for {
			n, err := r.store.DeleteOlderThan(ctx, time.Now().Add(-r.cfg.Retention))
			if err != nil {
				r.log.ErrorContext(ctx, "runlog: очистка журнала", "error", err)
			} else {
				r.log.InfoContext(ctx, "runlog: очистка журнала", "deleted", n)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

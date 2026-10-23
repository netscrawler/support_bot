// Package metrics содержит prometheus-метрики сервиса и сервер /metrics + /healthz.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var runBuckets = []float64{0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600}

var (
	ReportRuns = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_report_runs_total",
		Help: "Report runs by kind and status.",
	}, []string{"kind", "status"})

	ReportRunDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sbot_report_run_duration_seconds",
		Help:    "Report run duration by kind.",
		Buckets: runBuckets,
	}, []string{"kind"})

	CollectorDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "sbot_collector_duration_seconds",
		Help: "Collector duration by type.",
	}, []string{"type"})

	CollectorErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_collector_errors_total",
		Help: "Collector errors by type.",
	}, []string{"type"})

	DeliveryRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_delivery_retries_total",
		Help: "Delivery retries by operation.",
	}, []string{"op"})

	DeliveryFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_delivery_failures_total",
		Help: "Delivery failures by operation.",
	}, []string{"op"})

	SchedulerRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sbot_scheduler_running",
		Help: "1 if the scheduler is running.",
	})

	BuildInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sbot_build_info",
		Help: "Build version and commit.",
	}, []string{"version", "commit"})
)

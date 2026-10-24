// Package metrics содержит prometheus-метрики сервиса и сервер /metrics + /healthz.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Имена label'ов.
const (
	lReport = "report"
	lType   = "type"
	lReason = "reason"
	lOp     = "op"
	lStatus = "status"
)

var runBuckets = []float64{0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600}

var (
	ReportRuns = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_report_runs_total",
		Help: "Число запусков отчётов по виду, статусу и отчёту.",
	}, []string{"kind", lStatus, lReport})

	ReportRunDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sbot_report_run_duration_seconds",
		Help:    "Длительность запуска отчёта по виду и отчёту.",
		Buckets: runBuckets,
	}, []string{"kind", lReport})

	ReportLastSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sbot_report_last_success_timestamp_seconds",
		Help: "Unix-время последнего успешного запуска отчёта.",
	}, []string{lReport})

	CollectorDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sbot_collector_duration_seconds",
		Help:    "Длительность сбора данных карточки по типу источника, отчёту и карточке.",
		Buckets: runBuckets,
	}, []string{lType, lReport, "card"})

	CollectorErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_collector_errors_total",
		Help: "Число ошибок сбора данных по типу источника, отчёту, карточке и причине.",
	}, []string{lType, lReport, "card", lReason})

	DeliveryRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_delivery_retries_total",
		Help: "Число повторных попыток доставки по операции.",
	}, []string{lOp})

	DeliveryFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_delivery_failures_total",
		Help: "Число неудачных доставок по операции и причине.",
	}, []string{lOp, lReason})

	DeliverySent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_delivery_sent_total",
		Help: "Число успешных доставок по операции.",
	}, []string{lOp})

	DeliveryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sbot_delivery_duration_seconds",
		Help:    "Длительность доставки по операции.",
		Buckets: runBuckets,
	}, []string{lOp})

	DeliveryRecipient = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_delivery_recipient_total",
		Help: "Число доставок получателям по каналу, статусу и причине.",
	}, []string{"channel", lStatus, lReason})

	SourceHTTPRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_source_http_retries_total",
		Help: "Число повторных HTTP-запросов к источникам по хосту.",
	}, []string{"host"})

	SourceHTTPFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_source_http_failures_total",
		Help: "Число неудачных HTTP-запросов к источникам по хосту и причине.",
	}, []string{"host", lReason})

	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_http_requests_total",
		Help: "Число входящих HTTP-запросов по маршруту и коду ответа.",
	}, []string{"route", "code"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "sbot_http_request_duration_seconds",
		Help: "Длительность обработки входящих HTTP-запросов по маршруту.",
	}, []string{"route"})

	BotUpdates = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_bot_updates_total",
		Help: "Число обработанных обновлений бота по типу, команде и статусу.",
	}, []string{lType, "command", lStatus})

	BotUpdateDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "sbot_bot_update_duration_seconds",
		Help: "Длительность обработки обновления бота по типу.",
	}, []string{lType})

	LuaDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "sbot_lua_duration_seconds",
		Help: "Длительность выполнения Lua-скрипта.",
	}, []string{"script"})

	LuaErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sbot_lua_errors_total",
		Help: "Число ошибок выполнения Lua-скриптов по скрипту и причине.",
	}, []string{"script", lReason})

	SchedulerRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sbot_scheduler_running",
		Help: "1, если планировщик запущен.",
	})

	BuildInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sbot_build_info",
		Help: "Версия и коммит сборки.",
	}, []string{"version", "commit"})
)

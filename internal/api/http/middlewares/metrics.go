package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"support_bot/internal/metrics"
)

// Metrics считает запросы и их длительность по шаблону маршрута (r.Pattern),
// а не по URL, чтобы не плодить кардинальность.
func (mw *MW) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		metrics.HTTPRequests.WithLabelValues(r.Pattern, strconv.Itoa(rec.status)).
			Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Pattern).
			Observe(time.Since(start).Seconds())
	})
}

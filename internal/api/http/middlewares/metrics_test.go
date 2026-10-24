package middlewares

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"support_bot/internal/metrics"
	"support_bot/internal/pkg/httplib"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMW_Metrics_CountsByRoutePatternAndStatus(t *testing.T) {
	mw := NewMiddleware(slog.New(slog.DiscardHandler), 0)

	router := httplib.NewRouter()
	router.Use(mw.Metrics)
	router.Get("/report/{id}", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	router.Get("/fail", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ok := metrics.HTTPRequests.WithLabelValues("GET /report/{id}", "200")
	fail := metrics.HTTPRequests.WithLabelValues("GET /fail", "500")
	okBefore, failBefore := testutil.ToFloat64(ok), testutil.ToFloat64(fail)

	for _, path := range []string{"/report/42", "/fail"} {
		router.Mux().ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, path, http.NoBody),
		)
	}

	if got := testutil.ToFloat64(ok) - okBefore; got != 1 {
		t.Errorf("200 delta = %v, want 1", got)
	}

	if got := testutil.ToFloat64(fail) - failBefore; got != 1 {
		t.Errorf("500 delta = %v, want 1", got)
	}
}

package metrics

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func get(
	t *testing.T,
	ping func(context.Context) error,
	path string,
) (int, string) {
	t.Helper()

	ts := httptest.NewServer(NewServer("", ping, slog.Default()).srv.Handler)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = resp.Body.Close() }()

	b, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, string(b)
}

func TestHealthzOK(t *testing.T) {
	code, body := get(t, func(context.Context) error { return nil }, "/healthz")
	if code != 200 || body != "ok" {
		t.Fatalf("got %d %q", code, body)
	}
}

func TestHealthzError(t *testing.T) {
	code, body := get(
		t,
		func(context.Context) error { return errors.New("db down") },
		"/healthz",
	)
	if code != 503 || !strings.Contains(body, "db down") {
		t.Fatalf("got %d %q", code, body)
	}
}

func TestHealthzTimeout(t *testing.T) {
	start := time.Now()
	code, _ := get(t, func(ctx context.Context) error {
		<-ctx.Done()

		return ctx.Err()
	}, "/healthz")

	if code != 503 {
		t.Fatalf("got %d", code)
	}

	if d := time.Since(start); d < 1900*time.Millisecond || d > 3*time.Second {
		t.Fatalf("took %s", d)
	}
}

func TestMetricsBuildInfo(t *testing.T) {
	BuildInfo.WithLabelValues("test", "test").Set(1)

	code, body := get(t, func(context.Context) error { return nil }, "/metrics")
	if code != 200 || !strings.Contains(body, "sbot_build_info") {
		t.Fatalf("got %d, no sbot_build_info", code)
	}
}

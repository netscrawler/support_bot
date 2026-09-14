package middlewares

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMW_Gzip_CompressesWhenAccepted(t *testing.T) {
	mw := NewMiddleware(slog.Default(), 1024, "")

	handler := mw.Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if ce := rec.Header().Get("Content-Encoding"); ce != "gzip" {
		t.Fatalf("got Content-Encoding %q, want gzip", ce)
	}

	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("response body is not valid gzip: %v", err)
	}
	defer gr.Close()

	got, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read gzip body: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("got body %q, want %q", got, "hello world")
	}
}

func TestMW_Gzip_PassesThroughWhenNotAccepted(t *testing.T) {
	mw := NewMiddleware(slog.Default(), 1024, "")

	handler := mw.Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if ce := rec.Header().Get("Content-Encoding"); ce != "" {
		t.Fatalf("got Content-Encoding %q, want empty", ce)
	}
	if rec.Body.String() != "hello world" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "hello world")
	}
}

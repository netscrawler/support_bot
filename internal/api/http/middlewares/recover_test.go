package middlewares

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMW_RecoverMiddleware_ReturnsInternalServerErrorAndDoesNotLogHeaders(t *testing.T) {
	// Проверяем: паника в обработчике перехватывается, отдаётся 500,
	// а в лог не попадают заголовки запроса (Authorization и т.п.).
	var logBuf bytes.Buffer
	mw := NewMiddleware(slog.New(slog.NewTextHandler(&logBuf, nil)), 1024)

	handler := mw.RecoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/report", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if strings.Contains(logBuf.String(), "secret-token") {
		t.Fatalf("log output leaked Authorization header value: %s", logBuf.String())
	}
}

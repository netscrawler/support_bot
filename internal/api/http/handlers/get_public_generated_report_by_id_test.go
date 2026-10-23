package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"support_bot/internal/models"
	"testing"
)

type fakeProvider struct{ err error }

func (f fakeProvider) GenerateReport(context.Context, string) (models.Data, error) {
	return models.Data{}, f.err
}

func TestGetGeneratedReportByID_Errors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"expired", models.ErrLinkExpired, http.StatusGone},
		{"not found", models.ErrNotFound, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(
				fakeProvider{err: tt.err},
				slog.New(slog.NewTextHandler(io.Discard, nil)),
			)
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.SetPathValue("report_id", "123e4567-e89b-42d3-a456-426614174000")
			w := httptest.NewRecorder()

			h.GetGeneratedReportByID(w, r)

			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

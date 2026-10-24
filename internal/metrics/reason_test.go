package metrics

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"support_bot/internal/pkg/httplib"

	"github.com/mymmrac/telego/telegoapi"
)

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

func TestReason(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"canceled", context.Canceled, "canceled"},
		{"deadline", context.DeadlineExceeded, "timeout"},
		{
			"wrapped deadline",
			fmt.Errorf("wrap: %w", context.DeadlineExceeded),
			"timeout",
		},
		{"net timeout", timeoutError{}, "timeout"},
		{"telegram 403", &telegoapi.Error{ErrorCode: 403}, "http_4xx"},
		{"telegram 502", &telegoapi.Error{ErrorCode: 502}, "http_5xx"},
		{"httplib 404", httplib.NewHTTPError(404, "x"), "http_4xx"},
		{"other", errors.New("x"), "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reason(tt.err); got != tt.want {
				t.Errorf("Reason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReportCtx(t *testing.T) {
	if got := ReportFrom(context.Background()); got != "" {
		t.Errorf("ReportFrom(empty) = %q", got)
	}

	if got := ReportFrom(
		WithReport(context.Background(), "daily"),
	); got != "daily" {
		t.Errorf("ReportFrom() = %q, want daily", got)
	}
}

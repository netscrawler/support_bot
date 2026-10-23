package telegram

import (
	"context"
	"errors"
	"net"
	"support_bot/internal/pkg/retry"
	"testing"
	"time"

	"github.com/mymmrac/telego/telegoapi"
)

func TestClassify(t *testing.T) {
	apiErr := func(code, after int) error {
		e := &telegoapi.Error{ErrorCode: code}
		if after > 0 {
			e.Parameters = &telegoapi.ResponseParameters{RetryAfter: after}
		}

		return e
	}
	netErr := &net.OpError{Op: "dial", Err: errors.New("refused")}

	tests := []struct {
		name      string
		err       error
		wantCalls int   // вызовов fn при delays{0} и коротком контексте
		wantErr   error // что вернёт Do
		wantWait  bool  // ждал ли паузу After (упёрся в таймаут контекста)
	}{
		{"400 permanent", apiErr(400, 0), 1, nil, false},
		{"403 permanent", apiErr(403, 0), 1, nil, false},
		{"429 retry_after=3 ждёт 3s", apiErr(429, 3), 1, context.DeadlineExceeded, true},
		{"429 retry_after=120 permanent", apiErr(429, 120), 1, nil, false},
		{"сеть повторяется", netErr, 2, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			calls := 0
			got := retry.Do(ctx, retry.Delays{0}, nil, "t", func(context.Context) error {
				calls++

				return classify(tt.err)
			})

			if calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", calls, tt.wantCalls)
			}

			if tt.wantWait {
				if !errors.Is(got, tt.wantErr) {
					t.Errorf("err = %v, want %v", got, tt.wantErr)
				}

				return
			}

			if got != tt.err { //nolint:errorlint // исходная ошибка должна вернуться как есть
				t.Errorf("err = %v, want original %v", got, tt.err)
			}
		})
	}

	if classify(nil) != nil {
		t.Error("classify(nil) != nil")
	}
}

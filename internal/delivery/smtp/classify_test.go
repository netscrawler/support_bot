package smtp

import (
	"context"
	"errors"
	"fmt"
	"net/textproto"
	"support_bot/internal/pkg/retry"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		calls int
	}{
		{"550 permanent", &textproto.Error{Code: 550}, 1},
		{"421 retryable", &textproto.Error{Code: 421}, 2},
		{"wrapped 550", fmt.Errorf("add recipient: %w", &textproto.Error{Code: 550}), 1},
		{"plain error retryable", errors.New("boom"), 2},
		{"nil", nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			_ = retry.Do(
				context.Background(),
				retry.Delays{0},
				nil,
				"t",
				func(context.Context) error {
					calls++

					return classify(tt.err)
				},
			)

			if calls != tt.calls {
				t.Fatalf("calls = %d, want %d", calls, tt.calls)
			}
		})
	}
}

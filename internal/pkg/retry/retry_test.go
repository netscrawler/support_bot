package retry

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"support_bot/internal/metrics"
	"support_bot/internal/pkg/secret"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

var errBoom = errors.New("boom")

func TestDo_SuccessOnSecondAttempt(t *testing.T) {
	calls := 0
	err := Do(
		context.Background(),
		Delays{time.Millisecond},
		nil,
		"op",
		func(context.Context) error {
			calls++
			if calls == 1 {
				return errBoom
			}

			return nil
		},
	)

	if err != nil || calls != 2 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestDo_AllFail(t *testing.T) {
	delays := Delays{time.Millisecond, time.Millisecond}
	calls := 0
	err := Do(
		context.Background(),
		delays,
		nil,
		"op",
		func(context.Context) error {
			calls++

			return errBoom
		},
	)

	if !errors.Is(err, errBoom) || calls != len(delays)+1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestDo_Permanent(t *testing.T) {
	calls := 0
	err := Do(
		context.Background(),
		Delays{time.Millisecond},
		nil,
		"op",
		func(context.Context) error {
			calls++

			return Permanent(errBoom)
		},
	)

	if !errors.Is(err, errBoom) ||
		calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestDo_AfterOverridesDelay(t *testing.T) {
	calls := 0
	start := time.Now()
	err := Do(
		context.Background(),
		Delays{time.Hour},
		nil,
		"op",
		func(context.Context) error {
			calls++
			if calls == 1 {
				return After(errBoom, time.Millisecond)
			}

			return nil
		},
	)

	if err != nil || calls != 2 || time.Since(start) > time.Minute {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestDo_AfterUnwrappedWhenExhausted(t *testing.T) {
	err := Do(
		context.Background(),
		nil,
		nil,
		"op",
		func(context.Context) error {
			return After(errBoom, time.Hour)
		},
	)
	if err != errBoom { //nolint:errorlint // проверяем отсутствие обёртки
		t.Fatalf("err=%v", err)
	}
}

func TestDo_CancelDuringPause(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(10*time.Millisecond, cancel)

	start := time.Now()
	calls := 0
	err := Do(ctx, Delays{time.Hour}, nil, "op", func(context.Context) error {
		calls++

		return errBoom
	})

	if !errors.Is(err, context.Canceled) || calls != 1 ||
		time.Since(start) > time.Minute {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestDo_EmptyDelays(t *testing.T) {
	calls := 0
	err := Do(
		context.Background(),
		nil,
		nil,
		"op",
		func(context.Context) error {
			calls++

			return errBoom
		},
	)

	if !errors.Is(err, errBoom) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestDo_Metrics(t *testing.T) {
	retries := metrics.DeliveryRetries.WithLabelValues("metrics_op")
	failures := metrics.DeliveryFailures.WithLabelValues("metrics_op")
	r0, f0 := testutil.ToFloat64(retries), testutil.ToFloat64(failures)

	calls := 0

	err := Do(context.Background(), Delays{time.Millisecond}, nil, "metrics_op",
		func(context.Context) error {
			calls++
			if calls == 1 {
				return errBoom
			}

			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	if d := testutil.ToFloat64(retries) - r0; d != 1 {
		t.Fatalf("retries delta=%v", d)
	}

	if d := testutil.ToFloat64(failures) - f0; d != 0 {
		t.Fatalf("failures delta=%v", d)
	}
}

func TestDo_RedactsSecretInLog(t *testing.T) {
	secret.Register("retry-tok-777")

	var buf bytes.Buffer

	log := slog.New(slog.NewTextHandler(&buf, nil))
	calls := 0

	err := Do(
		context.Background(),
		Delays{time.Millisecond},
		log,
		"op",
		func(context.Context) error {
			calls++
			if calls == 1 {
				return errors.New("Get https://api/botretry-tok-777/x: EOF")
			}

			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, "retry-tok-777") || !strings.Contains(out, "***") {
		t.Fatalf("log=%s", out)
	}
}

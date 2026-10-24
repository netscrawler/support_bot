package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"support_bot/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

type fakeRT struct {
	err    error
	cancel context.CancelFunc
}

func (f fakeRT) RoundTrip(*http.Request) (*http.Response, error) {
	if f.cancel != nil {
		go func() {
			time.Sleep(10 * time.Millisecond)
			f.cancel()
		}()
	}

	if f.err != nil {
		return nil, f.err
	}

	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestRoundTripper_Metrics(t *testing.T) {
	const host = "src.test"

	retries := metrics.SourceHTTPRetries.WithLabelValues(host)
	failures := metrics.SourceHTTPFailures.WithLabelValues(host, "canceled")
	r0, f0 := testutil.ToFloat64(retries), testutil.ToFloat64(failures)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://"+host+"/x",
		http.NoBody,
	)

	_, err := NewRoundTripper(
		fakeRT{err: errors.New("boom"), cancel: cancel},
	).RoundTrip(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}

	if d := testutil.ToFloat64(retries) - r0; d != 1 {
		t.Fatalf("retries delta=%v", d)
	}

	if d := testutil.ToFloat64(failures) - f0; d != 1 {
		t.Fatalf("failures delta=%v", d)
	}

	r0, f0 = testutil.ToFloat64(retries), testutil.ToFloat64(failures)
	req, _ = http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://"+host+"/x",
		http.NoBody,
	)

	if _, err := NewRoundTripper(fakeRT{}).RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	if testutil.ToFloat64(retries) != r0 || testutil.ToFloat64(failures) != f0 {
		t.Fatal("counters changed on success")
	}
}

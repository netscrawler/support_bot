package retry

import (
	"net/http"
	"time"

	"support_bot/internal/metrics"
)

type RoundTripper struct {
	t http.RoundTripper
}

func NewRoundTripper(tr http.RoundTripper) RoundTripper {
	return RoundTripper{t: tr}
}

func (x RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	retry := 3
	delay := 15 * time.Second

	for r := 0; ; r++ {
		resp, err := x.t.RoundTrip(req)
		if err == nil {
			return resp, nil
		}

		if r >= retry {
			metrics.SourceHTTPFailures.WithLabelValues(req.URL.Host, metrics.Reason(err)).
				Inc()

			return resp, err
		}

		metrics.SourceHTTPRetries.WithLabelValues(req.URL.Host).Inc()

		select {
		case <-time.After(delay):
			delay *= 3
		case <-ctx.Done():
			metrics.SourceHTTPFailures.WithLabelValues(req.URL.Host, metrics.Reason(ctx.Err())).
				Inc()

			return &http.Response{}, ctx.Err()
		}
	}
}

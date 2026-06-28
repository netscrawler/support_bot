package metabase

import (
	"net/http"
	"time"
)

type readableRoundTripper struct {
	t http.RoundTripper
}

func newRetractileRoundTripper(tr http.RoundTripper) readableRoundTripper {
	return readableRoundTripper{t: tr}
}

func (x readableRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	retry := 3
	delay := 15 * time.Second

	for r := 0; ; r++ {
		resp, err := x.t.RoundTrip(req)
		if err == nil || r >= retry {
			return resp, err
		}

		select {
		case <-time.After(delay):
			delay = delay * 3
		case <-ctx.Done():
			return &http.Response{}, ctx.Err()
		}
	}
}

package retry

import (
	"net/http"
	"time"
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

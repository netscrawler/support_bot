package metrics

import (
	"context"
	"errors"
	"net"

	"support_bot/internal/pkg/httplib"

	"github.com/mymmrac/telego/telegoapi"
)

// Reason сводит ошибку к малокардинальной причине для label reason.
func Reason(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		return "timeout"
	}

	if apiErr, ok := errors.AsType[*telegoapi.Error](err); ok {
		return httpClass(apiErr.ErrorCode)
	}

	if httpErr, ok := errors.AsType[*httplib.HTTPError](err); ok {
		return httpClass(httpErr.Code)
	}

	return "other"
}

func httpClass(code int) string {
	switch {
	case code >= 400 && code < 500:
		return "http_4xx"
	case code >= 500 && code < 600:
		return "http_5xx"
	}

	return "other"
}

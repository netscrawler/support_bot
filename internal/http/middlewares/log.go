package middlewares

import (
	"log/slog"
	"net/http"
	httputils "support_bot/internal/http/utils"
	"time"
)

type LogMW struct {
	log *slog.Logger
}

func NewLogMW(log *slog.Logger) *LogMW {
	return &LogMW{log: log}
}

func (lm *LogMW) ServeHTTPFunc(next http.Handler) http.Handler {
	l := lm.log

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.InfoContext(
			r.Context(),
			"request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Any("headers", maskHeaders(r.Header)),
		)

		rw := httputils.NewResponseW(w)
		start := time.Now()

		defer func() {
			duration := time.Since(start)

			l.InfoContext(
				r.Context(),
				"response",
				slog.Int("status_code", rw.StatusCode()),
				slog.Int("size_bytes", rw.Size()),
				slog.Any("headers", maskHeaders(rw.Header())),
				slog.Duration("duration", duration),
			)
		}()

		next.ServeHTTP(rw, r)
	})
}

var sensitiveHeaders = map[string]struct{}{
	"Authorization": {},
	"Cookie":        {},
	"Set-Cookie":    {},
}

func maskHeaders(h http.Header) http.Header {
	out := http.Header{}

	for k, v := range h {
		if _, ok := sensitiveHeaders[k]; ok {
			out[k] = []string{"***"}

			continue
		}

		out[k] = v
	}

	return out
}

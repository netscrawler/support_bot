package middlewares

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter

	status      int
	body        bytes.Buffer
	maxBodySize int64
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if remaining := r.maxBodySize - int64(r.body.Len()); remaining > 0 {
		if int64(len(b)) > remaining {
			r.body.Write(b[:remaining])
		} else {
			r.body.Write(b)
		}
	}

	return r.ResponseWriter.Write(b)
}

func (mw *MW) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var loggedBody []byte

		if r.Body != nil {
			limited := io.LimitReader(r.Body, mw.maxBodySize)
			buf, err := io.ReadAll(limited)
			if err != nil {
				mw.log.ErrorContext(
					r.Context(),
					"Failed to read request body",
					slog.String("error", err.Error()),
				)
			}

			loggedBody = buf

			r.Body = struct {
				io.Reader
				io.Closer
			}{
				Reader: io.MultiReader(bytes.NewReader(buf), r.Body),
				Closer: r.Body,
			}
		}

		mw.log.InfoContext(r.Context(), "Incoming request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("body", string(loggedBody)),
		)

		rec := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
			maxBodySize:    mw.maxBodySize,
		}
		next.ServeHTTP(rec, r)

		stop := time.Now()
		mw.log.InfoContext(r.Context(), "Request completed",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("remote_addr", r.RemoteAddr),
			slog.Int("status", rec.status),
			slog.Int("response_body_size", rec.body.Len()),
			slog.Duration("duration", stop.Sub(start)),
		)
	})
}

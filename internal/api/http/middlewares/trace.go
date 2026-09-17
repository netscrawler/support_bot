package middlewares

import (
	"log/slog"
	"net/http"
	"support_bot/internal/pkg/logger"

	"github.com/google/uuid"
)

const TraceIDKey = "trace_id"

func (mw *MW) Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace := uuid.New().String()

		ctx := logger.AppendCtx(r.Context(), slog.String(TraceIDKey, trace))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package middlewares

import (
	"net/http"
	"support_bot/internal/pkg/logger"

	"github.com/google/uuid"
)

const Header = "X-Trace-ID"

func TraceMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(Header)

		if traceID == "" {
			traceID = uuid.NewString()
		}

		ctx := logger.WithTraceID(r.Context(), traceID)

		w.Header().Set(Header, traceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

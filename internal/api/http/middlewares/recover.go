package middlewares

import "net/http"

func (mw *MW) RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				mw.log.Error("panic recovered", "error", err, "request", r)

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

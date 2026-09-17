package middlewares

import "net/http"

func (mw *MW) RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Логируем только метод и путь: r целиком содержит заголовки
				// (Authorization/Cookie), которые нельзя писать в лог.
				mw.log.Error("panic recovered",
					"error", err,
					"method", r.Method,
					"url", r.URL.String(),
				)

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

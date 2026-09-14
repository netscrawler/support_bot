package middlewares

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"support_bot/internal/pkg/httplib"
)

func (mw *MW) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(mw.authToken)) != 1 {
			httplib.ErrUnauthorized.Write(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

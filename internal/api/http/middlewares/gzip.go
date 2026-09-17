package middlewares

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gw *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gw.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

// Gzip compresses the response body when the client sends
// Accept-Encoding: gzip.
func (mw *MW) Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")

		gw := gzip.NewWriter(w)
		defer gw.Close()

		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gw: gw}, r)
	})
}

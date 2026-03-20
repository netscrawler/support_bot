package http

import "net/http"

type Middleware func(http.Handler) http.Handler

type Router struct {
	mux    *http.ServeMux
	prefix string
	mw     []Middleware
}

func NewRouter(mux *http.ServeMux) *Router {
	return &Router{
		mux: mux,
	}
}

func (r *Router) Use(m ...Middleware) {
	r.mw = append(r.mw, m...)
}

func (r *Router) Group(prefix string) *Router {
	return &Router{
		mux:    r.mux,
		prefix: r.prefix + prefix,
		mw:     append([]Middleware{}, r.mw...),
	}
}

func (r *Router) Handle(method, path string, h http.HandlerFunc) {
	full := method + " " + r.prefix + path

	handler := chain(h, r.mw)

	r.mux.Handle(full, handler)

	if method != http.MethodOptions {
		r.mux.Handle("OPTIONS "+r.prefix+path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
	}
}

func (r *Router) GET(path string, h http.HandlerFunc) {
	r.Handle(http.MethodGet, path, h)
}

func (r *Router) POST(path string, h http.HandlerFunc) {
	r.Handle(http.MethodPost, path, h)
}

func (r *Router) PUT(path string, h http.HandlerFunc) {
	r.Handle(http.MethodPut, path, h)
}

func (r *Router) PATCH(path string, h http.HandlerFunc) {
	r.Handle(http.MethodPatch, path, h)
}

func (r *Router) DELETE(path string, h http.HandlerFunc) {
	r.Handle(http.MethodDelete, path, h)
}

func (r *Router) Handler() http.Handler {
	return r.mux
}

func chain(h http.Handler, mw []Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}

	return h
}

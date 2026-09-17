package httplib

import (
	"net/http"
	"slices"
)

type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	PATCH  HTTPMethod = "PATCH"
	DELETE HTTPMethod = "DELETE"
	HEAD   HTTPMethod = "HEAD"
)

// Middleware matches the signature already used by internal/api/httplib/middlewares
// (e.g. MW.LogRequest, MW.RecoverMiddleware, MW.Trace), so they plug in directly.
type Middleware func(http.Handler) http.Handler

// Chain wraps h with mw so that mw[0] runs first, then mw[1], ..., then h.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for _, m := range slices.Backward(mw) {
		h = m(h)
	}

	return h
}

// Router registers routes on a shared httplib.ServeMux. Groups derived via
// Group share the mux but get their own prefix and middleware chain, so
// middleware added inside a group never leaks back to its parent.
type Router struct {
	mux         *http.ServeMux
	prefix      string
	middlewares []Middleware
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

func NewRouterWithMux(mux *http.ServeMux) *Router {
	return &Router{mux: mux}
}

// Mux returns the underlying mux, e.g. to wrap it in global middleware
// (logging/recover/trace) that must also cover unmatched routes.
func (r *Router) Mux() *http.ServeMux {
	return r.mux
}

// Use appends middlewares applied to every route registered on this router,
// and inherited by any group derived from it, from this point on.
func (r *Router) Use(mw ...Middleware) {
	r.middlewares = append(r.middlewares, mw...)
}

// Group creates a scoped router sharing the same mux, with prefix and
// middleware chain inherited from the parent at the time Group is called.
func (r *Router) Group(prefix string, fn func(r *Router)) {
	child := &Router{
		mux:         r.mux,
		prefix:      r.prefix + prefix,
		middlewares: append([]Middleware(nil), r.middlewares...),
	}
	fn(child)
}

func (r *Router) Handle(method HTTPMethod, pattern string, handler http.Handler) {
	r.mux.Handle(string(method)+" "+r.prefix+pattern, Chain(handler, r.middlewares...))
}

func (r *Router) HandleFunc(method HTTPMethod, pattern string, handler http.HandlerFunc) {
	r.Handle(method, pattern, handler)
}

func (r *Router) Get(pattern string, handler http.HandlerFunc) {
	r.HandleFunc(GET, pattern, handler)
}

func (r *Router) Post(pattern string, handler http.HandlerFunc) {
	r.HandleFunc(POST, pattern, handler)
}

func (r *Router) Put(pattern string, handler http.HandlerFunc) {
	r.HandleFunc(PUT, pattern, handler)
}

func (r *Router) Patch(pattern string, handler http.HandlerFunc) {
	r.HandleFunc(PATCH, pattern, handler)
}

func (r *Router) Delete(pattern string, handler http.HandlerFunc) {
	r.HandleFunc(DELETE, pattern, handler)
}

func (r *Router) Head(pattern string, handler http.HandlerFunc) {
	r.HandleFunc(HEAD, pattern, handler)
}

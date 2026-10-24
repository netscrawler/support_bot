package httplib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func markerMiddleware(name string, order *[]string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*order = append(*order, name)

			next.ServeHTTP(w, r)
		})
	}
}

func TestRouter_Get_ServesRegisteredRoute(t *testing.T) {
	r := NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	rec := httptest.NewRecorder()
	r.Mux().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", http.NoBody))

	if rec.Code != http.StatusOK || rec.Body.String() != "pong" {
		t.Fatalf(
			"got status=%d body=%q, want 200 pong",
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestRouter_Use_RunsMiddlewareInRegistrationOrder(t *testing.T) {
	var order []string

	r := NewRouter()
	r.Use(markerMiddleware("first", &order), markerMiddleware("second", &order))
	r.Get("/x", func(_ http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
	})

	rec := httptest.NewRecorder()
	r.Mux().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", http.NoBody))

	want := []string{"first", "second", "handler"}
	if len(order) != len(want) {
		t.Fatalf("got order=%v, want %v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("got order=%v, want %v", order, want)
		}
	}
}

func TestRouter_Group_InheritsPrefixAndParentMiddleware(t *testing.T) {
	var order []string

	r := NewRouter()
	r.Use(markerMiddleware("global", &order))

	r.Group("/admin", func(gr *Router) {
		gr.Use(markerMiddleware("admin-only", &order))
		gr.Get("/users", func(_ http.ResponseWriter, _ *http.Request) {
			order = append(order, "handler")
		})
	})

	// Route is mounted under the group prefix.
	rec := httptest.NewRecorder()
	r.Mux().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status=%d, want 200", rec.Code)
	}

	want := []string{"global", "admin-only", "handler"}
	if len(order) != len(want) {
		t.Fatalf("got order=%v, want %v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("got order=%v, want %v", order, want)
		}
	}

	// Middleware added inside the group must not leak back to the parent router.
	order = nil
	r.Get("/root", func(_ http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
	})

	rec = httptest.NewRecorder()
	r.Mux().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/root", http.NoBody))

	want = []string{"global", "handler"}
	if len(order) != len(want) {
		t.Fatalf(
			"got order=%v, want %v (admin-only middleware leaked into parent)",
			order,
			want,
		)
	}
}

func TestChain_WrapsOuterToInner(t *testing.T) {
	var order []string

	h := Chain(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			order = append(order, "handler")
		}),
		markerMiddleware("outer", &order),
		markerMiddleware("inner", &order),
	)

	h.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/", http.NoBody),
	)

	want := []string{"outer", "inner", "handler"}
	if len(order) != len(want) {
		t.Fatalf("got order=%v, want %v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("got order=%v, want %v", order, want)
		}
	}
}

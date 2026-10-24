package httpcoll

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"support_bot/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSecret struct{ value, host string }

type fakeSecrets map[string]fakeSecret

// local — секрет, привязанный к хосту тестового сервера без порта.
func local(v string) fakeSecret { return fakeSecret{v, "127.0.0.1"} }

func (f fakeSecrets) Get(
	_ context.Context,
	name string,
) (string, string, error) {
	v, ok := f[name]
	if !ok {
		return "", "", errors.New("secret not found")
	}

	return v.value, v.host, nil
}

// newServer поднимает сервер и считает запросы.
func newServer(
	t *testing.T,
	h http.HandlerFunc,
) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32

	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			h(w, r)
		}),
	)
	t.Cleanup(srv.Close)

	return srv, &calls
}

func jsonReply(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(body))
	}
}

func newColl(t *testing.T, cfgs map[string]Config, s SecretGetter) *Collector {
	t.Helper()

	c, err := New(cfgs, s, slog.Default())
	require.NoError(t, err)

	return c
}

func TestConfigSourceURL(t *testing.T) {
	t.Parallel()

	var got *http.Request

	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		jsonReply(`[{"id":1}]`)(w, r)
	})
	c := newColl(t, map[string]Config{"billing": {BaseURL: srv.URL + "/"}}, nil)

	rows, err := c.Fetch(
		t.Context(),
		"billing",
		"/api/items?x=1",
		map[string]string{"from": "2026-10-01"},
	)
	require.NoError(t, err)
	assert.Equal(t, []map[string]any{{"id": float64(1)}}, rows)
	assert.Equal(t, "/api/items", got.URL.Path)
	assert.Equal(t, "1", got.URL.Query().Get("x"))
	assert.Equal(t, "2026-10-01", got.URL.Query().Get("from"))
	assert.Equal(t, http.MethodGet, got.Method)
}

func TestAbsoluteURL(t *testing.T) {
	t.Parallel()

	var path string

	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		jsonReply(`{"a":1}`)(w, r)
	})
	c := newColl(t, nil, nil)

	_, err := c.Fetch(t.Context(), srv.URL+"/v1/stats", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "/v1/stats", path)
}

func TestUnknownSource(t *testing.T) {
	t.Parallel()

	srv, calls := newServer(t, jsonReply(`[]`))
	c := newColl(t, map[string]Config{"billing": {BaseURL: srv.URL}}, nil)

	_, err := c.Fetch(t.Context(), "billingg", "/x", nil)
	require.ErrorContains(t, err, "billingg")
	assert.Zero(t, calls.Load())
}

func TestHeadersOverride(t *testing.T) {
	t.Parallel()

	var accept, extra string

	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		accept, extra = r.Header.Get("Accept"), r.Header.Get("X-Extra")
		jsonReply(`[]`)(w, r)
	})
	c := newColl(t, map[string]Config{"s": {
		BaseURL: srv.URL,
		Headers: map[string]string{"Accept": "text/csv", "X-Extra": "1"},
	}}, nil)

	_, err := c.FetchCard(t.Context(), models.Card{
		CardUUID: "s",
		Options:  []byte(`{"headers":{"Accept":"application/json"}}`),
	})
	require.NoError(t, err)
	assert.Equal(t, "application/json", accept)
	assert.Equal(t, "1", extra)
}

func TestBadOptions(t *testing.T) {
	t.Parallel()

	srv, calls := newServer(t, jsonReply(`[]`))
	c := newColl(t, nil, nil)

	for opts, want := range map[string]string{
		`{"fromat":"csv"}`:   "fromat",
		`{"format":"html"}`:  "html",
		`{"auth":"digest"}`:  "digest",
		`{"delimiter":";;"}`: "delimiter",
		`{"timeout":"soon"}`: "soon",
	} {
		_, err := c.FetchCard(
			t.Context(),
			models.Card{CardUUID: srv.URL, Options: []byte(opts)},
		)
		require.ErrorContains(t, err, want)
	}

	assert.Zero(t, calls.Load())

	_, err := c.FetchCard(
		t.Context(),
		models.Card{CardUUID: srv.URL, Options: []byte(`{}`)},
	)
	require.NoError(t, err)
}

func TestAuthFromConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cfg   Config
		check func(t *testing.T, r *http.Request)
	}{
		{
			"bearer",
			Config{Auth: "bearer", Token: "t0ken"},
			func(t *testing.T, r *http.Request) {
				t.Helper()
				assert.Equal(t, "Bearer t0ken", r.Header.Get("Authorization"))
			},
		},
		{
			"basic",
			Config{Auth: "basic", User: "u", Password: "pass"},
			func(t *testing.T, r *http.Request) {
				t.Helper()

				u, p, ok := r.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "u", u)
				assert.Equal(t, "pass", p)
			},
		},
		{
			"header",
			Config{Auth: "header", AuthHeader: "X-Api-Key", Token: "k3y!"},
			func(t *testing.T, r *http.Request) {
				t.Helper()
				assert.Equal(t, "k3y!", r.Header.Get("X-Api-Key"))
				assert.Empty(t, r.Header.Get("Authorization"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got *http.Request

			srv, _ := newServer(
				t,
				func(w http.ResponseWriter, r *http.Request) {
					got = r
					jsonReply(`[]`)(w, r)
				},
			)

			tt.cfg.BaseURL = srv.URL

			_, err := newColl(
				t,
				map[string]Config{"s": tt.cfg},
				nil,
			).Fetch(t.Context(), "s", "", nil)
			require.NoError(t, err)
			tt.check(t, got)
		})
	}
}

func TestAuthFromSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, options, value string
		check                func(t *testing.T, r *http.Request)
	}{
		{"default bearer", ``, "t0k", func(t *testing.T, r *http.Request) {
			t.Helper()
			assert.Equal(t, "Bearer t0k", r.Header.Get("Authorization"))
		}},
		{
			"basic",
			`{"auth":"basic"}`,
			"user:pa:ss",
			func(t *testing.T, r *http.Request) {
				t.Helper()

				u, p, _ := r.BasicAuth()
				assert.Equal(t, "user", u)
				assert.Equal(t, "pa:ss", p)
			},
		},
		{
			"header",
			`{"auth":"header","auth_header":"X-Api-Key"}`,
			"k3y",
			func(t *testing.T, r *http.Request) {
				t.Helper()
				assert.Equal(t, "k3y", r.Header.Get("X-Api-Key"))
				assert.Empty(t, r.Header.Get("Authorization"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got *http.Request

			srv, _ := newServer(
				t,
				func(w http.ResponseWriter, r *http.Request) {
					got = r
					jsonReply(`[]`)(w, r)
				},
			)
			// Секрет карточки перекрывает конфиг.
			c := newColl(t, map[string]Config{"s": {
				BaseURL: srv.URL, Auth: "bearer", Token: "config-token",
			}}, fakeSecrets{"sec": local(tt.value)})

			_, err := c.FetchCard(t.Context(), models.Card{
				CardUUID: "s", Secret: "sec", Options: []byte(tt.options),
			})
			require.NoError(t, err)
			tt.check(t, got)
		})
	}
}

func TestSecretUnavailable(t *testing.T) {
	t.Parallel()

	srv, calls := newServer(t, jsonReply(`[]`))

	_, err := newColl(
		t,
		nil,
		fakeSecrets{},
	).FetchCard(t.Context(), models.Card{CardUUID: srv.URL, Secret: "missing"})
	require.ErrorContains(t, err, "missing")

	_, err = newColl(
		t,
		nil,
		nil,
	).FetchCard(t.Context(), models.Card{CardUUID: srv.URL, Secret: "sec"})
	require.ErrorContains(t, err, "sec")

	assert.Zero(t, calls.Load())
}

func TestNon2xx(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)

		_, _ = w.Write([]byte("internal error " + strings.Repeat("x", 1000)))
	})

	_, err := newColl(
		t,
		nil,
		nil,
	).Fetch(t.Context(), srv.URL+"/p", "", map[string]string{"api_key": "qwerty"})
	require.ErrorContains(t, err, "500")
	require.ErrorContains(t, err, "internal error")
	assert.NotContains(t, err.Error(), "qwerty")
	assert.Less(t, len(err.Error()), 700)
}

func TestSecretRedactedInError(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)

		_, _ = w.Write([]byte("bad token t0k and cfg-token"))
	})
	c := newColl(t, map[string]Config{"s": {
		BaseURL: srv.URL, Auth: "bearer", Token: "cfg-token",
	}}, fakeSecrets{"sec": local("t0k")})

	_, err := c.FetchCard(
		t.Context(),
		models.Card{CardUUID: "s", Secret: "sec"},
	)
	require.ErrorContains(t, err, "401")
	require.ErrorContains(t, err, "bad token *** and ***")
}

func TestBodyLimit(t *testing.T) {
	old := maxBody

	maxBody = 10
	t.Cleanup(func() { maxBody = old })

	srv, _ := newServer(t, jsonReply(`[1,2,3,4,5,6,7,8]`))

	_, err := newColl(t, nil, nil).Fetch(t.Context(), srv.URL, "", nil)
	require.ErrorContains(t, err, "exceeds 50 MB")
}

func TestUnknownContentType(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		_, _ = w.Write([]byte("<html>"))
	})
	c := newColl(t, nil, nil)

	_, err := c.Fetch(t.Context(), srv.URL, "", nil)
	require.ErrorContains(t, err, "text/html")

	rows, err := c.FetchCard(
		t.Context(),
		models.Card{CardUUID: srv.URL, Options: []byte(`{"format":"csv"}`)},
	)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestNewValidation(t *testing.T) {
	t.Parallel()

	for name, cfg := range map[string]Config{
		"empty url":      {},
		"relative url":   {BaseURL: "/api"},
		"ftp":            {BaseURL: "ftp://x.local"},
		"no host":        {BaseURL: "http://"},
		"bad auth":       {BaseURL: "http://x.local", Auth: "digest"},
		"header no name": {BaseURL: "http://x.local", Auth: "header"},
	} {
		_, err := New(map[string]Config{"s": cfg}, nil, slog.Default())
		require.Error(t, err, name)
	}

	_, err := New(
		map[string]Config{"s": {BaseURL: "https://x.local", Auth: "basic"}},
		nil,
		slog.Default(),
	)
	require.NoError(t, err)
}

func TestCrossHostRedirectKeepsSecret(t *testing.T) {
	t.Parallel()

	evil, evilCalls := newServer(t, jsonReply(`[]`))
	srv, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/steal", http.StatusFound)
	})

	c := newColl(t, nil, fakeSecrets{"k": local("k3y")})

	_, err := c.FetchCard(t.Context(), models.Card{
		CardUUID: srv.URL,
		Options:  []byte(`{"auth":"header","auth_header":"X-Api-Key"}`),
		Secret:   "k",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "302")
	assert.Zero(t, evilCalls.Load())
}

func TestErrorChainKept(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := newColl(t, nil, nil).Fetch(ctx, srv.URL, "", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSecretHostBinding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    func(srvHost string) string
		wantErr bool
	}{
		{"other host", func(string) string { return "billing.local" }, true},
		{
			"same host with port",
			strings.ToUpper,
			false,
		},
		{"same host without port", func(h string) string {
			return strings.Split(h, ":")[0]
		}, false},
		{"other port", func(string) string { return "127.0.0.1:1" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, calls := newServer(t, jsonReply(`[]`))
			u, err := url.Parse(srv.URL)
			require.NoError(t, err)

			c := newColl(t, nil, fakeSecrets{
				"sec": {"t0k", tt.host(u.Host)},
			})

			_, err = c.FetchCard(t.Context(), models.Card{
				CardUUID: srv.URL, Secret: "sec",
			})

			if tt.wantErr {
				require.ErrorContains(
					t, err,
					`secret "sec" is bound to host`,
				)
				require.ErrorContains(t, err, u.Host)
				assert.Zero(t, calls.Load())

				return
			}

			require.NoError(t, err)
			assert.Equal(t, int32(1), calls.Load())
		})
	}
}

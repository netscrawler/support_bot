package prometheus

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildSources(t *testing.T) {
	gr := map[string]GrafanaConfig{
		"ops": {
			URL:     "https://g.example/",
			Token:   "gt",
			Timeout: 5 * time.Second,
		},
		"ops0": {URL: "https://g0.example", Token: "gt"},
	}

	bad := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			"оба варианта",
			Config{URL: "http://p", Grafana: "ops", DatasourceUID: "u"},
			"либо url",
		},
		{"ни одного", Config{}, "ни url"},
		{
			"grafana без uid",
			Config{Grafana: "ops"},
			"и grafana, и datasource_uid",
		},
		{
			"uid без grafana",
			Config{DatasourceUID: "u"},
			"и grafana, и datasource_uid",
		},
		{
			"нет grafana",
			Config{Grafana: "nope", DatasourceUID: "u"},
			`grafana "nope"`,
		},
		{
			"bearer и basic",
			Config{URL: "http://p", BearerToken: "t", BasicAuth: &BasicAuth{}},
			"взаимоисключающие",
		},
	}
	for _, tc := range bad {
		_, err := buildSources(map[string]Config{"bad": tc.cfg}, gr)
		require.Error(t, err, tc.name)
		require.Contains(t, err.Error(), `"bad"`, tc.name)
		require.Contains(t, err.Error(), tc.want, tc.name)
	}

	srcs, err := buildSources(map[string]Config{
		"plain":  {URL: "http://p/"},
		"bearer": {URL: "http://p", BearerToken: "bt", Timeout: time.Second},
		"basic": {
			URL:       "http://p",
			BasicAuth: &BasicAuth{User: "u", Password: "pw"},
		},
		"proxy":  {Grafana: "ops", DatasourceUID: "abc"},
		"proxy2": {Grafana: "ops", DatasourceUID: "abc", Timeout: time.Minute},
		"proxy0": {Grafana: "ops0", DatasourceUID: "abc"},
	}, gr)
	require.NoError(t, err)

	req := func(name string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		if a := srcs[name].auth; a != nil {
			a(r)
		}

		return r
	}

	require.Equal(t, "http://p", srcs["plain"].baseURL)
	require.Empty(t, req("plain").Header.Get("Authorization"))
	require.Equal(t, defaultTimeout, srcs["plain"].timeout)
	require.Equal(t, "Bearer bt", req("bearer").Header.Get("Authorization"))
	require.Equal(t, time.Second, srcs["bearer"].timeout)

	u, p, ok := req("basic").BasicAuth()
	require.True(t, ok)
	require.Equal(t, "u", u)
	require.Equal(t, "pw", p)

	require.Equal(
		t,
		"https://g.example/api/datasources/proxy/uid/abc",
		srcs["proxy"].baseURL,
	)
	require.Equal(t, "Bearer gt", req("proxy").Header.Get("Authorization"))
	require.Equal(t, 5*time.Second, srcs["proxy"].timeout)
	require.Equal(t, time.Minute, srcs["proxy2"].timeout)
	require.Equal(t, defaultTimeout, srcs["proxy0"].timeout)
}

func TestBuildRequest(t *testing.T) {
	path, v, err := buildRequest(`sum(rate(x{s="${svc}"}[${rng}])) $raw`,
		map[string]string{"svc": "api", "rng": "5m", "extra": "1"})
	require.NoError(t, err)
	require.Equal(t, "/api/v1/query", path)
	require.Equal(t, url.Values{"query": {`sum(rate(x{s="api"}[5m])) $raw`}}, v)

	_, _, err = buildRequest("up{e=${env}}", map[string]string{})
	require.ErrorContains(t, err, "env")

	_, v, err = buildRequest("up", map[string]string{"time": "123"})
	require.NoError(t, err)
	require.Equal(t, url.Values{"query": {"up"}, "time": {"123"}}, v)

	path, v, err = buildRequest(
		"up",
		map[string]string{
			"start": "1",
			"end":   "2",
			"step":  "1h",
			"time":  "9",
			"x":     "y",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/query_range", path)
	require.Equal(
		t,
		url.Values{
			"query": {"up"},
			"start": {"1"},
			"end":   {"2"},
			"step":  {"1h"},
		},
		v,
	)

	_, _, err = buildRequest("up", map[string]string{"start": "1"})
	require.ErrorContains(t, err, "end, step")
}

func newTestCollector(
	t *testing.T,
	h http.HandlerFunc,
	cfg Config,
	gr map[string]GrafanaConfig,
) *Collector {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	if cfg.URL == "" && cfg.Grafana == "" {
		cfg.URL = srv.URL
	}

	for k, g := range gr {
		g.URL = srv.URL
		gr[k] = g
	}

	c, err := New(
		map[string]Config{"main": cfg},
		gr,
		slog.New(slog.DiscardHandler),
	)
	require.NoError(t, err)

	return c
}

func respond(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = io.WriteString(w, body)
	}
}

func TestFetchVector(t *testing.T) {
	var got *http.Request

	c := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		respond(
			`{"status":"success","warnings":["w"],"data":{"resultType":"vector","result":[
			{"metric":{"__name__":"up","job":"api"},"value":[1700000000,"1"]},
			{"metric":{"job":"nan"},"value":[1700000000,"NaN"]},
			{"metric":{"job":"hist"},"histogram":[1700000000,{}]}]}}`,
		)(
			w,
			r,
		)
	}, Config{}, nil)

	rows, err := c.Fetch(
		context.Background(),
		"main",
		"up",
		map[string]string{"time": "42"},
	)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/query", got.URL.Path)
	require.Equal(t, "up", got.URL.Query().Get("query"))
	require.Equal(t, "42", got.URL.Query().Get("time"))
	require.Equal(t, []map[string]any{
		{"metric": "up", "job": "api", "value": float64(1)},
		{"job": "nan", "value": nil},
		{"job": "hist", "value": nil},
	}, rows)
}

func TestFetchMatrix(t *testing.T) {
	var got *http.Request

	c := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		respond(`{"status":"success","data":{"resultType":"matrix","result":[
			{"metric":{"__name__":"up"},"values":[[1700000000,"1"],[1700003600.5,"+Inf"]]}]}}`)(w, r)
	}, Config{}, nil)

	rows, err := c.Fetch(context.Background(), "main", "up",
		map[string]string{"start": "1", "end": "2", "step": "1h"})
	require.NoError(t, err)
	require.Equal(t, "/api/v1/query_range", got.URL.Path)
	require.Equal(t, "1h", got.URL.Query().Get("step"))
	require.Equal(t, []map[string]any{
		{
			"metric":    "up",
			"timestamp": "2023-11-14T22:13:20Z",
			"value":     float64(1),
		},
		{"metric": "up", "timestamp": "2023-11-14T23:13:20Z", "value": nil},
	}, rows)
}

func TestFetchErrors(t *testing.T) {
	c := newTestCollector(
		t,
		respond(
			`{"status":"error","errorType":"bad_data","error":"parse error"}`,
		),
		Config{},
		nil,
	)
	_, err := c.Fetch(context.Background(), "main", "up{", nil)
	require.ErrorContains(t, err, "bad_data: parse error")

	_, err = c.Fetch(context.Background(), "nope", "up", nil)
	require.EqualError(t, err, `unknown prometheus source "nope"`)

	c = newTestCollector(
		t,
		respond(
			`{"status":"success","data":{"resultType":"scalar","result":[1,"2"]}}`,
		),
		Config{},
		nil,
	)
	_, err = c.Fetch(context.Background(), "main", "1+1", nil)
	require.ErrorContains(t, err, "scalar")

	c = newTestCollector(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)

		_, _ = io.WriteString(w, "<html>bad gateway</html>")
	}, Config{}, nil)
	_, err = c.Fetch(context.Background(), "main", "up", nil)
	require.ErrorContains(t, err, "502")

	c = newTestCollector(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)

		_, _ = io.WriteString(w, `{"message":"invalid API key"}`)
	}, Config{}, nil)
	_, err = c.Fetch(context.Background(), "main", "up", nil)
	require.ErrorContains(t, err, "HTTP 401")
	require.ErrorContains(t, err, "invalid API key")
}

func TestFetchAuth(t *testing.T) {
	var got *http.Request

	h := func(w http.ResponseWriter, r *http.Request) {
		got = r
		respond(
			`{"status":"success","data":{"resultType":"vector","result":[]}}`,
		)(
			w,
			r,
		)
	}

	c := newTestCollector(t, h, Config{Grafana: "ops", DatasourceUID: "abc"},
		map[string]GrafanaConfig{"ops": {Token: "gt"}})
	_, err := c.Fetch(context.Background(), "main", "up", nil)
	require.NoError(t, err)
	require.Equal(
		t,
		"/api/datasources/proxy/uid/abc/api/v1/query",
		got.URL.Path,
	)
	require.Equal(t, "Bearer gt", got.Header.Get("Authorization"))

	c = newTestCollector(
		t,
		h,
		Config{BasicAuth: &BasicAuth{User: "u", Password: "p"}},
		nil,
	)
	_, err = c.Fetch(context.Background(), "main", "up", nil)
	require.NoError(t, err)

	u, p, ok := got.BasicAuth()
	require.True(t, ok)
	require.Equal(t, "u", u)
	require.Equal(t, "p", p)
}

func TestNewValidates(t *testing.T) {
	_, err := New(map[string]Config{"x": {}}, nil, slog.Default())
	require.ErrorContains(t, err, `"x"`)
}

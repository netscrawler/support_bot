package grafana

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"support_bot/internal/collector/prometheus"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const rulesBody = `{"status":"success","data":{"groups":[
{"name":"g1","file":"folderA","rules":[
 {"name":"HighCPU","state":"firing","health":"ok","lastEvaluation":"2026-10-23T10:00:00Z","labels":{"team":"ops"},
  "alerts":[
   {"labels":{"alertname":"HighCPU","host":"a"},"annotations":{"summary":"hot"},"state":"Alerting","activeAt":"2026-10-23T09:00:00Z","value":"[ var='B' value=1 ]"},
   {"labels":{"host":"b","state":"x"},"annotations":{},"state":"Pending","activeAt":"2026-10-23T09:30:00Z","value":"v2"}]},
 {"name":"Idle","state":"inactive","health":"ok","lastEvaluation":"0001-01-01T00:00:00Z","labels":{},
  "alerts":[{"labels":{"host":"n"},"annotations":{},"state":"Normal (MissingSeries)","activeAt":"2026-10-23T09:00:00Z","value":""}]}]},
{"name":"g2","file":"folderB","rules":[
 {"name":"Disk","state":"pending","health":"ok","lastEvaluation":"2026-10-23T10:00:00Z","labels":{},
  "alerts":[{"labels":{"host":"c"},"annotations":{},"state":"Pending","activeAt":"0001-01-01T00:00:00Z","value":"v3"}]}]}]}}`

func newTestCollector(
	t *testing.T,
	h http.HandlerFunc,
) (*Collector, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)

	c := New(map[string]prometheus.GrafanaConfig{
		"g": {URL: srv.URL + "/", Token: "secret", Timeout: 5 * time.Second},
	}, slog.Default())

	return c, &calls
}

func rulesHandler(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(rulesBody)) }

func TestPathAndBearer(t *testing.T) {
	c, _ := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/prometheus/grafana/api/v1/rules", r.URL.Path)
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		rulesHandler(w, r)
	})

	_, err := c.Fetch(context.Background(), "g", "rules", nil)
	require.NoError(t, err)
}

func TestRules(t *testing.T) {
	c, _ := newTestCollector(t, rulesHandler)

	rows, err := c.Fetch(context.Background(), "g", "rules", nil)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, map[string]any{
		"folder": "folderA", "group": "g1", "rule": "HighCPU", "state": "firing",
		"health": "ok", "last_evaluation": "2026-10-23T10:00:00Z",
		"active_instances": 2, "team": "ops",
	}, rows[0])
	require.Equal(t, 0, rows[1]["active_instances"])
	require.Nil(t, rows[1]["last_evaluation"])
}

func TestActive(t *testing.T) {
	c, _ := newTestCollector(t, rulesHandler)

	rows, err := c.Fetch(context.Background(), "g", "active", nil)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, map[string]any{
		"alertname": "HighCPU", "state": "Alerting", "active_at": "2026-10-23T09:00:00Z",
		"value": "[ var='B' value=1 ]", "folder": "folderA", "group": "g1",
		"host": "a", "annotation_summary": "hot",
	}, rows[0])
	// служебная колонка state перезаписывает одноимённый label
	require.Equal(t, "Pending", rows[1]["state"])
	require.Nil(t, rows[2]["active_at"])
}

func TestFilters(t *testing.T) {
	c, _ := newTestCollector(t, rulesHandler)
	ctx := context.Background()

	rows, err := c.Fetch(
		ctx,
		"g",
		"active",
		map[string]string{"state": "Pending", "folder": "folderB"},
	)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "Disk", rows[0]["alertname"])

	rows, err = c.Fetch(ctx, "g", "rules", map[string]string{"alertname": "Idle"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "Idle", rows[0]["rule"])

	rows, err = c.Fetch(ctx, "g", "rules", map[string]string{"group": "g2"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestValidationBeforeRequest(t *testing.T) {
	c, calls := newTestCollector(t, rulesHandler)
	ctx := context.Background()

	_, err := c.Fetch(ctx, "g", "rules", map[string]string{"limit": "5"})
	require.ErrorContains(t, err, "limit")

	_, err = c.Fetch(ctx, "g", "history", map[string]string{"folder": "x"})
	require.ErrorContains(t, err, "folder")

	_, err = c.Fetch(ctx, "g", "nope", nil)
	require.ErrorContains(t, err, "active, rules, history")

	require.Zero(t, calls.Load())
}

func TestHistory(t *testing.T) {
	c, _ := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/annotations", r.URL.Path)
		q := r.URL.Query()
		require.Equal(t, "alert", q.Get("type"))
		require.Equal(t, "1792749600000", q.Get("from"))
		require.Equal(t, "1792753200000", q.Get("to"))
		require.Equal(t, "10000", q.Get("limit"))
		_, _ = w.Write([]byte(`[
{"time":1792749600000,"timeEnd":0,"alertName":"A","prevState":"Normal","newState":"Alerting","text":"t1"},
{"time":1792749700000,"timeEnd":1792749800000,"alertName":"","prevState":"Alerting","newState":"Normal","text":"B {alertname=B, team=x} - "}]`))
	})

	rows, err := c.Fetch(context.Background(), "g", "history",
		map[string]string{"from": "2026-10-23T10:00:00Z", "to": "1792753200000"})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "2026-10-23T10:00:00Z", rows[0]["time"])
	require.Nil(t, rows[0]["time_end"])
	require.Equal(t, "2026-10-23T10:03:20Z", rows[1]["time_end"])
	require.Equal(t, "Alerting", rows[0]["new_state"])
	require.Equal(t, "A", rows[0]["alertname"])
	require.Equal(t, "B", rows[1]["alertname"]) // unified alerting: имя только в text

	rows, err = c.Fetch(context.Background(), "g", "history",
		map[string]string{"from": "2026-10-23T10:00:00Z", "to": "1792753200000", "alertname": "B"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestHistoryBadParams(t *testing.T) {
	c, calls := newTestCollector(t, rulesHandler)

	_, err := c.Fetch(context.Background(), "g", "history", map[string]string{"from": "вчера"})
	require.ErrorContains(t, err, "from")

	_, err = c.Fetch(context.Background(), "g", "history", map[string]string{"limit": "0"})
	require.ErrorContains(t, err, "limit")

	require.Zero(t, calls.Load())
}

func TestHTTPError(t *testing.T) {
	c, _ := newTestCollector(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"access denied"}`))
	})

	for _, mode := range []string{"rules", "history"} {
		_, err := c.Fetch(context.Background(), "g", mode, nil)
		require.ErrorContains(t, err, "403")
		require.ErrorContains(t, err, "access denied")
		require.NotContains(t, err.Error(), "secret")
	}
}

func TestStatusError(t *testing.T) {
	c, _ := newTestCollector(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"boom"}`))
	})

	_, err := c.Fetch(context.Background(), "g", "rules", nil)
	require.ErrorContains(t, err, "bad_data: boom")

	c, _ = newTestCollector(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})

	_, err = c.Fetch(context.Background(), "g", "rules", nil)
	require.ErrorContains(t, err, "не rules API")
}

func TestUnknownGrafana(t *testing.T) {
	c, _ := newTestCollector(t, rulesHandler)

	_, err := c.Fetch(context.Background(), "zzz", "rules", nil)
	require.ErrorContains(t, err, `unknown grafana "zzz"`)
}

package zabbix

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestCollector(t *testing.T, h http.HandlerFunc) (*Collector, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)

	c, err := New(map[string]Config{
		"z": {URL: srv.URL + "/", Token: "secret", Timeout: 5 * time.Second},
	}, slog.Default())
	require.NoError(t, err)

	return c, &calls
}

func reply(result string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":` + result + `,"id":1}`))
	}
}

func fetch(c *Collector, query string, params map[string]string) ([]map[string]any, error) {
	return c.Fetch(context.Background(), "z", query, params)
}

func TestRequestShape(t *testing.T) {
	c, calls := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api_jsonrpc.php", r.URL.Path)
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Equal(t, "application/json-rpc", r.Header.Get("Content-Type"))

		b, _ := io.ReadAll(r.Body)
		require.JSONEq(t,
			`{"jsonrpc":"2.0","method":"problem.get","params":{"severities":[4,5]},"id":1}`,
			string(b))
		reply("[]")(w, r)
	})

	_, err := fetch(c, `{"method":"problem.get","params":{"severities":[4,5]}}`, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, calls.Load())
}

func TestRejectedWithoutRequest(t *testing.T) {
	c, calls := newTestCollector(t, reply("[]"))

	_, err := fetch(c, `{"method":"host.delete","params":["10084"]}`, nil)
	require.ErrorContains(t, err, "host.delete")

	// Zabbix берёт только первые два сегмента: host.delete.get выполнил бы host.delete.
	for _, m := range []string{"host.delete.get", ".get", "host..get", "HOST.GET"} {
		_, err = fetch(c, `{"method":"`+m+`","params":{}}`, nil)
		require.ErrorContains(t, err, "запрещён", m)
	}

	_, err = fetch(c, `{"method":"host.get","params":{},"extra":1}`, nil)
	require.Error(t, err)

	_, err = fetch(c, `{"method":"host.get"} {"x":1}`, nil)
	require.Error(t, err)

	_, err = fetch(c, `{"params":{}}`, nil)
	require.Error(t, err)

	require.Zero(t, calls.Load())
}

func TestSubstitution(t *testing.T) {
	var got map[string]any

	c, _ := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(b, &got))
		reply("[]")(w, r)
	})

	_, err := fetch(c, `{"method":"host.get","params":{"groupids":["${group}"]}}`,
		map[string]string{"group": "15"})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"groupids": []any{"15"}}, got["params"])

	_, err = fetch(c, `{"method":"host.get","params":{"search":{"name":"${q}"}}}`,
		map[string]string{"q": `a", "x": "1`})
	require.NoError(t, err)
	require.Equal(t,
		map[string]any{"search": map[string]any{"name": `a", "x": "1`}}, got["params"])
}

func TestMissingPlaceholder(t *testing.T) {
	c, calls := newTestCollector(t, reply("[]"))

	_, err := fetch(c, `{"method":"host.get","params":{"host":"${host}"}}`, nil)
	require.ErrorContains(t, err, "host")
	require.Zero(t, calls.Load())
}

func TestNumberLiteralKept(t *testing.T) {
	c, _ := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		require.Contains(t, string(b), `"time_from":1729700000`)
		reply("[]")(w, r)
	})

	_, err := fetch(c, `{"method":"problem.get","params":{"time_from":1729700000}}`, nil)
	require.NoError(t, err)
}

func TestFlattenNested(t *testing.T) {
	c, _ := newTestCollector(t, reply(
		`[{"eventid":"1","severity":"4","hosts":[{"host":"db1"}],"tags":[{"tag":"svc","value":"api"}]}]`,
	))

	rows, err := fetch(c, `{"method":"problem.get","params":{}}`, nil)
	require.NoError(t, err)
	require.Equal(t, []map[string]any{{
		"eventid": "1", "severity": "4", "hosts.0.host": "db1",
		"tags.0.tag": "svc", "tags.0.value": "api",
	}}, rows)
}

func TestScalarObjectEmptyAndNumber(t *testing.T) {
	q := `{"method":"host.get","params":{}}`

	c, _ := newTestCollector(t, reply(`"42"`))
	rows, err := fetch(c, q, nil)
	require.NoError(t, err)
	require.Equal(t, []map[string]any{{"result": "42"}}, rows)

	c, _ = newTestCollector(t, reply(`{"a":{"b":"c"}}`))
	rows, err = fetch(c, q, nil)
	require.NoError(t, err)
	require.Equal(t, []map[string]any{{"a.b": "c"}}, rows)

	c, _ = newTestCollector(t, reply(`[]`))
	rows, err = fetch(c, q, nil)
	require.NoError(t, err)
	require.Empty(t, rows)

	c, _ = newTestCollector(t, reply(`[{"n":5}]`))
	rows, err = fetch(c, q, nil)
	require.NoError(t, err)
	require.Equal(t, float64(5), rows[0]["n"])
}

func TestRPCError(t *testing.T) {
	c, _ := newTestCollector(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(
			[]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params.",` +
				`"data":"Invalid parameter \"/\": unexpected parameter \"foo\"."},"id":1}`),
		)
	})

	_, err := fetch(c, `{"method":"problem.get","params":{"foo":1}}`, nil)
	require.ErrorContains(t, err, "Invalid params.")
	require.ErrorContains(t, err, "unexpected parameter")
	require.NotContains(t, err.Error(), "secret")
}

func TestHTTP502(t *testing.T) {
	c, _ := newTestCollector(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>Bad Gateway</html>"))
	})

	_, err := fetch(c, `{"method":"host.get","params":{}}`, nil)
	require.ErrorContains(t, err, "502")
}

func TestUnknownSource(t *testing.T) {
	c, _ := newTestCollector(t, reply("[]"))

	_, err := c.Fetch(context.Background(), "nope", `{"method":"host.get"}`, nil)
	require.ErrorContains(t, err, "nope")
}

func TestNewValidation(t *testing.T) {
	_, err := New(map[string]Config{"prod": {Token: "t"}}, slog.Default())
	require.ErrorContains(t, err, `zabbix "prod": не задан url`)

	_, err = New(map[string]Config{"prod": {URL: "http://x"}}, slog.Default())
	require.ErrorContains(t, err, `zabbix "prod": не задан token`)
}

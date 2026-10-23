package appmetrica

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoSendsRealToken(t *testing.T) {
	var got string

	srv := httptest.NewServer(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = r.Header.Get("Authorization")
		}),
	)
	defer srv.Close()

	c := NewCollector(&Config{OAuthToken: "am-secret"}, slog.Default())
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		srv.URL,
		http.NoBody,
	)
	require.NoError(t, err)

	resp, err := c.do(req)
	require.NoError(t, err)

	_ = resp.Body.Close()

	require.Equal(t, "OAuth am-secret", got)
}

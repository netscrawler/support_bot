package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchSendsRealToken(t *testing.T) {
	var got string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := New(Config{AuthToken: "jira-secret", JiraHost: srv.URL}).Fetch(t.Context(), "x", nil)
	require.NoError(t, err)
	require.Equal(t, "Bearer jira-secret", got)
}

package httplib

import (
	"net/http/httptest"
	"strings"
	"testing"
)

type decodeTarget struct {
	Name string `json:"name"`
}

func TestDecode_ParsesValidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"alice"}`))

	v, herr := Decode[decodeTarget](req, 1024)
	if herr != nil {
		t.Fatalf("unexpected error: %v", herr)
	}
	if v.Name != "alice" {
		t.Fatalf("got name %q, want alice", v.Name)
	}
}

func TestDecode_MalformedJSON_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{not json`))

	_, herr := Decode[decodeTarget](req, 1024)
	if herr == nil {
		t.Fatal("expected an error, got nil")
	}
	if herr.Code != 400 {
		t.Fatalf("got code %d, want 400", herr.Code)
	}
}

func TestDecode_OversizedBody_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(
		"POST",
		"/",
		strings.NewReader(`{"name":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`),
	)

	_, herr := Decode[decodeTarget](req, 5)
	if herr == nil {
		t.Fatal("expected an error, got nil")
	}
	if herr.Code != 400 {
		t.Fatalf("got code %d, want 400", herr.Code)
	}
}

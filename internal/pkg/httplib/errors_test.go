package httplib

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestHTTPError_Error_ReturnsMessage(t *testing.T) {
	err := NewHTTPError(400, "bad input")

	if got := err.Error(); got != "bad input" {
		t.Fatalf("got %q, want %q", got, "bad input")
	}
}

func TestHTTPError_Write_SetsStatusAndJSONBody(t *testing.T) {
	err := NewHTTPError(404, "not found")

	rec := httptest.NewRecorder()
	err.Write(rec)

	if rec.Code != 404 {
		t.Fatalf("got status %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json", ct)
	}

	var body HTTPError
	if decErr := json.Unmarshal(rec.Body.Bytes(), &body); decErr != nil {
		t.Fatalf("failed to decode body: %v", decErr)
	}
	if body.Code != 404 || body.Message != "not found" {
		t.Fatalf("got body %+v, want {Code:404 Message:not found}", body)
	}
}

func TestDefaultHTTPErrors_HaveExpectedCodes(t *testing.T) {
	cases := []struct {
		err  *HTTPError
		code int
	}{
		{ErrBadRequest, 400},
		{ErrUnauthorized, 401},
		{ErrForbidden, 403},
		{ErrNotFound, 404},
		{ErrConflict, 409},
		{ErrInternal, 500},
	}

	for _, c := range cases {
		if c.err.Code != c.code {
			t.Errorf("got code %d, want %d", c.err.Code, c.code)
		}
		if c.err.Message == "" {
			t.Errorf("code %d: expected a non-empty explanatory message", c.code)
		}
	}
}

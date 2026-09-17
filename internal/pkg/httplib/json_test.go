package httplib

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSON_WritesStatusAndEncodesMap(t *testing.T) {
	rec := httptest.NewRecorder()

	RespondJSON(rec, 201, JSON{"ok": true, "id": 42.0})

	if rec.Code != 201 {
		t.Fatalf("got status %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["ok"] != true || body["id"] != 42.0 {
		t.Fatalf("got body %+v, want {ok:true id:42}", body)
	}
}

func TestJSON_EncodesArbitraryStruct(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	rec := httptest.NewRecorder()
	RespondJSON(rec, 200, payload{Name: "alice"})

	var body payload
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Name != "alice" {
		t.Fatalf("got name %q, want alice", body.Name)
	}
}

func TestRespondJSON_WithIndent_PrettyPrints(t *testing.T) {
	rec := httptest.NewRecorder()

	RespondJSON(rec, 200, JSON{"ok": true}, WithIndent("", "  "))

	want := "{\n  \"ok\": true\n}\n"
	if rec.Body.String() != want {
		t.Fatalf("got body %q, want %q", rec.Body.String(), want)
	}
}

func TestRespondJSON_WithEscapeHTMLFalse_LeavesHTMLCharsUnescaped(t *testing.T) {
	rec := httptest.NewRecorder()

	RespondJSON(rec, 200, JSON{"url": "a&b"}, WithEscapeHTML(false))

	if !strings.Contains(rec.Body.String(), "a&b") {
		t.Fatalf("got body %q, want unescaped a&b", rec.Body.String())
	}
}

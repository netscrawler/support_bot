package httplib

import (
	"encoding/json"
	"net/http"
)

// JSON is a shorthand for building an ad-hoc RespondJSON body without a named struct,
// e.g. httplib.RespondJSON(w, 200, httplib.JSON{"ok": true}).
type JSON map[string]any

// JSONOption configures the json.Encoder used by RespondJSON.
type JSONOption func(*json.Encoder)

func WithIndent(prefix, indent string) JSONOption {
	return func(enc *json.Encoder) { enc.SetIndent(prefix, indent) }
}

func WithEscapeHTML(escape bool) JSONOption {
	return func(enc *json.Encoder) { enc.SetEscapeHTML(escape) }
}

// RespondJSON writes v as a RespondJSON response with the given status code.
func RespondJSON(w http.ResponseWriter, code int, v any, opts ...JSONOption) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	enc := json.NewEncoder(w)
	for _, opt := range opts {
		opt(enc)
	}

	return enc.Encode(v)
}

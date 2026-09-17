package httplib

import (
	"encoding/json"
	"io"
	"net/http"
)

// Decode reads and JSON-decodes the request body into T, capped at maxBytes.
func Decode[T any](r *http.Request, maxBytes int64) (T, *HTTPError) {
	var v T

	buf, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	if err != nil {
		return v, NewHTTPError(http.StatusBadRequest, "failed to read request body: "+err.Error())
	}

	if int64(len(buf)) > maxBytes {
		return v, NewHTTPError(http.StatusBadRequest, "request body exceeds maximum allowed size")
	}

	if err := json.Unmarshal(buf, &v); err != nil {
		return v, NewHTTPError(http.StatusBadRequest, "invalid request body: "+err.Error())
	}

	return v, nil
}

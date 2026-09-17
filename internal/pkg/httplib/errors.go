package httplib

import (
	"encoding/json"
	"net/http"
)

// HTTPError is an error that knows how to render itself as a RespondJSON HTTP
// response.
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{Code: code, Message: message}
}

func (e *HTTPError) Error() string {
	return e.Message
}

// Write sets the response status to e.Code and writes e as a RespondJSON body.
func (e *HTTPError) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Code)
	json.NewEncoder(w).Encode(e)
}

var (
	ErrBadRequest = NewHTTPError(
		http.StatusBadRequest,
		"the request could not be understood or was missing required parameters",
	)
	ErrUnauthorized = NewHTTPError(
		http.StatusUnauthorized,
		"authentication is required and has failed or has not been provided",
	)
	ErrForbidden = NewHTTPError(
		http.StatusForbidden,
		"you do not have permission to access this resource",
	)
	ErrNotFound = NewHTTPError(http.StatusNotFound, "the requested resource could not be found")
	ErrConflict = NewHTTPError(
		http.StatusConflict,
		"the request conflicts with the current state of the resource",
	)
	ErrInternal = NewHTTPError(
		http.StatusInternalServerError,
		"an unexpected error occurred while processing the request",
	)
)

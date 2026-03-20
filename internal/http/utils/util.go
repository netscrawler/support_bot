package httputils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"support_bot/internal/http/errorz"
)

func UnmarshalFor[V any](data []byte) (V, error) {
	var v V
	err := json.Unmarshal(data, &v)
	if err != nil {
		return v, fmt.Errorf("%w: %w", errorz.ErrUnmarshallError, err)
	}
	return v, nil
}

type ResponseW struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
	size   int
}

func NewResponseW(w http.ResponseWriter) *ResponseW {
	return &ResponseW{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (r *ResponseW) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	n, err := r.ResponseWriter.Write(body)
	if n > 0 {
		r.body.Write(body[:n])
		r.size += n
	}

	return n, err
}

func (r *ResponseW) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseW) StatusCode() int {
	return r.status
}

func (r *ResponseW) Body() string {
	return r.body.String()
}

func (r *ResponseW) Size() int {
	return r.size
}

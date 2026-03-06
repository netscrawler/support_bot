package errorz

import (
	"errors"
	"net/http"
)

var ErrValidationError = errors.New("validation error")

var ErrUnmarshallError = errors.New("unmarshaling error")

var ErrInvalidRequest = ClientErr{
	Code: http.StatusBadRequest,
	Desc: "Invalid request",
}

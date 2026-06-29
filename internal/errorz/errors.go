package errorz

import "errors"

var (
	ErrNotFound     = errors.New("ErrNotFound")
	errInternal     = errors.New("InternalError")
	errAlreadyExist = errors.New("AlreadyExist")
)

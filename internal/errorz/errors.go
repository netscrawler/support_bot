package errorz

import "errors"

var (
	ErrNotFound     = errors.New("ErrNotFound")
	ErrInternal     = errors.New("ErrInternal")
	ErrAlreadyExist = errors.New("ErrAlreadyExist")
)

package models

import "errors"

var (
	ErrNotFound = errors.New("ErrNotFound")
	ErrInternal = errors.New("InternalError")
)

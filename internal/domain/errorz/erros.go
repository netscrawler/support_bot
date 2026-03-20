package errorz

import "errors"

var ErrUserNotFound = errors.New("user not found")

var ErrInternalServer = errors.New("internal server error")

var ErrInvalidToken = errors.New("invalid token")

var ErrTokenExpired = errors.New("token expired")

var ErrInvalidPassword = errors.New("invalid password")

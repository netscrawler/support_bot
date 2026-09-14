package middlewares

import "log/slog"

type MW struct {
	log         *slog.Logger
	maxBodySize int64

	authToken string
}

func NewMiddleware(log *slog.Logger, maxBodySize int64, authToken string) *MW {
	return &MW{
		log:         log,
		maxBodySize: maxBodySize,
		authToken:   authToken,
	}
}

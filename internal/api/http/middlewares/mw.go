package middlewares

import "log/slog"

type MW struct {
	log         *slog.Logger
	maxBodySize int64
}

func NewMiddleware(log *slog.Logger, maxBodySize int64) *MW {
	return &MW{
		log:         log,
		maxBodySize: maxBodySize,
	}
}

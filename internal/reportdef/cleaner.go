package reportdef

import (
	"context"
	"log/slog"
	"time"

	"support_bot/internal/store"
)

const DefaultCleanupInterval = time.Hour

// Cleaner periodically removes expired report_definitions rows.
type Cleaner struct {
	store    *store.ReportDefinitionStore
	log      *slog.Logger
	interval time.Duration
}

func NewCleaner(s *store.ReportDefinitionStore, log *slog.Logger) *Cleaner {
	return &Cleaner{
		store:    s,
		log:      log.With(slog.Any("module", "reportdef.cleaner")),
		interval: DefaultCleanupInterval,
	}
}

func (c *Cleaner) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.log.InfoContext(
					ctx,
					"context canceled, cleaner stopped",
					slog.Any("err", ctx.Err()),
				)

				return
			case <-ticker.C:
				n, err := c.store.CleanupExpired(ctx)
				if err != nil {
					c.log.ErrorContext(
						ctx,
						"cleanup expired report definitions failed",
						slog.Any("err", err),
					)

					continue
				}

				if n > 0 {
					c.log.InfoContext(
						ctx,
						"cleaned up expired report definitions",
						slog.Any("count", n),
					)
				}
			}
		}
	}()
}

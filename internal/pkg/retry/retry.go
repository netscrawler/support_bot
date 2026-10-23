package retry

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/metrics"
	"support_bot/internal/pkg/secret"
	"time"
)

// Delays — паузы между попытками Do; попыток = len+1.
type Delays []time.Duration

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

type afterError struct {
	err error
	d   time.Duration
}

func (e afterError) Error() string { return e.err.Error() }
func (e afterError) Unwrap() error { return e.err }

// Permanent помечает ошибку как неповторяемую.
func Permanent(err error) error {
	if err == nil {
		return nil
	}

	return permanentError{err}
}

// After помечает ошибку как повторяемую через d (вместо штатной паузы).
func After(err error, d time.Duration) error {
	if err == nil {
		return nil
	}

	return afterError{err, d}
}

// Do вызывает fn, пока она не вернёт nil, Permanent-ошибку или не кончатся паузы.
func Do(
	ctx context.Context,
	delays Delays,
	log *slog.Logger,
	op string,
	fn func(context.Context) error,
) error {
	if log == nil {
		log = slog.Default()
	}

	for i := 0; ; i++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		if p, ok := errors.AsType[permanentError](err); ok {
			metrics.DeliveryFailures.WithLabelValues(op).Inc()

			return p.err
		}

		a, isAfter := errors.AsType[afterError](err)
		if isAfter {
			err = a.err
		}

		if ctx.Err() != nil {
			return err
		}

		if i >= len(delays) {
			metrics.DeliveryFailures.WithLabelValues(op).Inc()

			return err
		}

		d := delays[i]
		if isAfter {
			d = a.d
		}

		metrics.DeliveryRetries.WithLabelValues(op).Inc()
		log.WarnContext(
			ctx,
			"retry",
			"op",
			op,
			"attempt",
			i+2,
			"delay",
			d,
			"error",
			secret.Redact(err.Error()),
		)

		timer := time.NewTimer(d)
		select {
		case <-ctx.Done():
			timer.Stop()

			return ctx.Err()
		case <-timer.C:
		}
	}
}

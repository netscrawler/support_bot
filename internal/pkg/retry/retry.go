package retry

import (
	"context"
	"errors"
	"log/slog"
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
			return p.err
		}

		a, isAfter := errors.AsType[afterError](err)
		if isAfter {
			err = a.err
		}

		if i >= len(delays) || ctx.Err() != nil {
			return err
		}

		d := delays[i]
		if isAfter {
			d = a.d
		}

		log.WarnContext(ctx, "retry", "op", op, "attempt", i+2, "delay", d, "error", err)

		timer := time.NewTimer(d)
		select {
		case <-ctx.Done():
			timer.Stop()

			return ctx.Err()
		case <-timer.C:
		}
	}
}

package telegram

import (
	"errors"
	"support_bot/internal/pkg/retry"
	"time"

	"github.com/mymmrac/telego/telegoapi"
)

const maxRetryAfter = 60

// classify размечает ошибку Telegram API для retry.Do: что не лечится повтором — Permanent,
// флуд-контроль — After, остальное (сеть, 5xx) повторяется как есть.
func classify(err error) error {
	apiErr, ok := errors.AsType[*telegoapi.Error](err)
	if err == nil || !ok {
		return err
	}

	switch apiErr.ErrorCode {
	case 400, 401, 403, 404:
		return retry.Permanent(err)
	case 429:
		if apiErr.Parameters == nil {
			return err
		}

		if apiErr.Parameters.RetryAfter > maxRetryAfter {
			return retry.Permanent(err)
		}

		return retry.After(err, time.Duration(apiErr.Parameters.RetryAfter)*time.Second)
	}

	return err
}

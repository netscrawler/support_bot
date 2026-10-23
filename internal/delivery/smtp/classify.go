package smtp

import (
	"errors"
	"net/textproto"
	"support_bot/internal/pkg/retry"
)

// classify: постоянные ответы SMTP (5xx) не повторяются.
func classify(err error) error {
	var tpErr *textproto.Error
	if errors.As(err, &tpErr) && tpErr.Code >= 500 {
		return retry.Permanent(err)
	}

	return err
}

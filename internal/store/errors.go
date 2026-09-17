package store

import (
	"errors"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5"
)

// translateNoRows maps pgx's no-rows sentinel to the domain-level
// models.ErrNotFound at the store boundary — every Store method that can
// return "not found" calls this on its query error before returning, so no
// caller outside internal/store ever needs to check pgx.ErrNoRows directly.
func translateNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrNotFound
	}

	return err
}

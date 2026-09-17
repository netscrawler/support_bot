package store

import (
	"errors"
	"support_bot/internal/models"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestTranslateNoRows_MapsPgxErrNoRows(t *testing.T) {
	err := translateNoRows(pgx.ErrNoRows)
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("translateNoRows(pgx.ErrNoRows) = %v, want models.ErrNotFound", err)
	}
}

func TestTranslateNoRows_PassesThroughOtherErrors(t *testing.T) {
	other := errors.New("connection reset")
	err := translateNoRows(other)
	if !errors.Is(err, other) {
		t.Fatalf("translateNoRows(other) = %v, want %v unchanged", err, other)
	}
}

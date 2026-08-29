//go:build !duckdb

package duck

import (
	"context"
	"errors"
	"support_bot/internal/models"
)

var ErrUnavailable = errors.New("duckdb not available on this build")

type DB struct{}

func New() (*DB, error) {
	return nil, ErrUnavailable
}

func (d *DB) LoadDataFromMapSlice(_ context.Context, _ models.Dataset) error {
	return ErrUnavailable
}

func (d *DB) ExecuteQuery(_ context.Context, _ string) ([]map[string]any, error) {
	return nil, ErrUnavailable
}

func (d *DB) Close() error {
	return ErrUnavailable
}

func (d *DB) InsertData(_ context.Context, _ string, _ []map[string]any) error {
	return ErrUnavailable
}

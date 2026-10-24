package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

const (
	testPublicID = "11111111-1111-1111-1111-111111111111"
)

func tstz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func TestReportStore_GetByPublicID_Expiry(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name       string
		expires    *time.Time
		wantErr    error
		wantGetRow bool
	}{
		{"no expiry reaches report", nil, models.ErrNotFound, true},
		{"future reaches report", &future, models.ErrNotFound, true},
		{"past is expired", &past, models.ErrLinkExpired, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock.NewPool() error = %v", err)
			}
			defer pool.Close()

			pool.ExpectQuery("select report_id, expires_at from public_reports").
				WithArgs(pgxmock.AnyArg()).
				WillReturnRows(pgxmock.NewRows([]string{"report_id", "expires_at"}).
					AddRow(int64(1), tstz(tt.expires)))

			if tt.wantGetRow {
				// ErrNoRows из GetByID доказывает, что запрос отчёта выполнен.
				pool.ExpectQuery("select r.id, r.name").
					WithArgs(int64(1)).
					WillReturnError(pgx.ErrNoRows)
			}

			s := NewReportStoreForTest(sqlcgen.New(pool))

			_, err = s.GetByPublicID(context.Background(), testPublicID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetByPublicID() error = %v, want %v", err, tt.wantErr)
			}

			if err := pool.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})
	}
}

func TestReportStore_CreatePublicLink_UnknownReport(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("select r.id, r.name").
		WithArgs("nope").
		WillReturnError(pgx.ErrNoRows)

	s := NewReportStoreForTest(sqlcgen.New(pool))

	if _, err := s.CreatePublicLink(
		context.Background(),
		"nope",
		time.Hour,
	); !errors.Is(
		err,
		models.ErrNotFound,
	) {
		t.Fatalf("CreatePublicLink() error = %v, want ErrNotFound", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations (no insert expected): %v", err)
	}
}

func TestReportStore_ExpirePublicLink(t *testing.T) {
	tests := []struct {
		name        string
		exists      bool
		wantErr     error
		wantExpired bool
	}{
		{"missing link", false, models.ErrNotFound, false},
		{"already inactive", true, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock.NewPool() error = %v", err)
			}
			defer pool.Close()

			pool.ExpectExec("update public_reports set expires_at = now()").
				WithArgs(pgxmock.AnyArg()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 0))

			get := pool.ExpectQuery("select report_id, expires_at from public_reports").
				WithArgs(pgxmock.AnyArg())
			if tt.exists {
				get.WillReturnRows(
					pgxmock.NewRows([]string{"report_id", "expires_at"}).
						AddRow(int64(1), tstz(nil)),
				)
			} else {
				get.WillReturnError(pgx.ErrNoRows)
			}

			s := NewReportStoreForTest(sqlcgen.New(pool))

			expired, err := s.ExpirePublicLink(
				context.Background(),
				testPublicID,
			)
			if !errors.Is(err, tt.wantErr) || expired != tt.wantExpired {
				t.Fatalf("ExpirePublicLink() = %v, %v; want %v, %v",
					expired, err, tt.wantExpired, tt.wantErr)
			}
		})
	}
}

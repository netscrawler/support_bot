package models_test

import (
	"testing"
	"time"

	"support_bot/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodName_Range(t *testing.T) {
	t.Parallel()

	loc := time.UTC
	at := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	}

	mar := time.Date(2026, 3, 10, 15, 4, 5, 0, loc)
	jan := time.Date(2026, 1, 15, 12, 0, 0, 0, loc)
	leap := time.Date(2028, 3, 1, 12, 0, 0, 0, loc)

	tests := []struct {
		name     string
		p        models.PeriodName
		now      time.Time
		from, to time.Time
	}{
		{"today", models.PeriodToday, mar, at(2026, 3, 10), at(2026, 3, 11)},
		{
			"yesterday",
			models.PeriodYesterday,
			mar,
			at(2026, 3, 9),
			at(2026, 3, 10),
		},
		{"week", models.PeriodWeek, mar, at(2026, 3, 3), at(2026, 3, 10)},
		{"month", models.PeriodMonth, mar, at(2026, 2, 1), at(2026, 3, 1)},
		{
			"month january",
			models.PeriodMonth,
			jan,
			at(2025, 12, 1),
			at(2026, 1, 1),
		},
		{"week leap", models.PeriodWeek, leap, at(2028, 2, 23), at(2028, 3, 1)},
		{
			"yesterday leap",
			models.PeriodYesterday,
			leap,
			at(2028, 2, 29),
			at(2028, 3, 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			from, to, err := tt.p.Range(tt.now)
			require.NoError(t, err)
			assert.True(t, tt.from.Equal(from), "from: %s", from)
			assert.True(t, tt.to.Equal(to), "to: %s", to)
		})
	}

	_, _, err := models.PeriodName("bogus").Range(time.Now())
	require.Error(t, err)
}

func TestParsePeriod(t *testing.T) {
	t.Parallel()

	p, err := models.ParsePeriod("")
	require.NoError(t, err)
	assert.Equal(t, models.PeriodToday, p)

	p, err = models.ParsePeriod("week")
	require.NoError(t, err)
	assert.Equal(t, models.PeriodWeek, p)

	_, err = models.ParsePeriod("bogus")
	require.Error(t, err)
}

func TestPeriodCodes(t *testing.T) {
	t.Parallel()

	all := []models.PeriodName{
		models.PeriodToday,
		models.PeriodYesterday,
		models.PeriodWeek,
		models.PeriodMonth,
	}

	for _, p := range all {
		got, ok := models.PeriodByCode(p.Code())
		assert.True(t, ok)
		assert.Equal(t, p, got)
	}

	_, ok := models.PeriodByCode("x")
	assert.False(t, ok)
}

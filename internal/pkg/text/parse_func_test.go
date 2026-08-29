package text_test

import (
	"fmt"
	"support_bot/internal/pkg/text"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteTemplate(t *testing.T) {
	t.Parallel()

	t.Run("date templates", func(t *testing.T) {
		t.Parallel()

		tmpl := `Отчёт за {{ now | yesterday | formatDateShort }}`

		want := fmt.Sprintf("Отчёт за %s", time.Now().AddDate(0, 0, -1).Format("02.01.2006"))

		got, err := text.ExecuteTemplate(tmpl, nil)

		require.NoError(t, err)

		assert.Equal(t, want, got)
	})

	t.Run("date month", func(t *testing.T) {
		t.Parallel()

		tmpl := `Отчёт за {{ now | lastMonth | formatRuMonthYear}}`

		// month names as produced by formatRuMonthYear
		months := []string{
			"январь", "февраль", "март", "апрель", "май", "июнь",
			"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
		}

		last := time.Now().AddDate(0, -1, 0)
		want := fmt.Sprintf("Отчёт за %s %d", months[int(last.Month())-1], last.Year())

		got, err := text.ExecuteTemplate(tmpl, nil)

		require.NoError(t, err)

		assert.Equal(t, want, got)
	})
}

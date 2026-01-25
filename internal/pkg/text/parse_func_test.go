package text_test

import (
	"fmt"
	"testing"
	"time"

	"support_bot/internal/pkg/text"

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

		want := "Отчёт за декабрь 2025"

		got, err := text.ExecuteTemplate(tmpl, nil)

		require.NoError(t, err)

		assert.Equal(t, want, got)
	})
}

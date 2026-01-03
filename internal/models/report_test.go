package models_test

import (
	"testing"

	"support_bot/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCron(t *testing.T) {
	t.Parallel()

	t.Run("valid cron", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("* * * * *")

		require.NoError(t, err)
		assert.Equal(t, models.Cron("* * * * *"), c, "cron must be equal to '* * * * *'")
	})

	t.Run("valid cron with minutes", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("5 * * * *")

		require.NoError(t, err)
		assert.Equal(t, models.Cron("5 * * * *"), c, "cron must be equal to '5 * * * *'")
	})

	t.Run("valid cron with min and hours", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("5 1 * * *")

		require.NoError(t, err)
		assert.Equal(t, models.Cron("5 1 * * *"), c, "cron must be equal to '5 1 * * *'")
	})

	t.Run("valid cron with /", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("5 0/10 * * *")

		require.NoError(t, err)
		assert.Equal(t, models.Cron("5 0/10 * * *"), c, "cron must be equal to '5 0/10 * * *'")
	})

	t.Run("invalid cron", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		_, err := models.NewCron("* * *")

		if assert.Error(t, err) {
			assert.Equal(t, models.ErrInvalidCron, err, "expected error")
		}
	})
}

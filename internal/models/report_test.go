package models_test

import (
	"support_bot/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCron(t *testing.T) {
	t.Parallel()

	t.Run("valid cron", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("* * * * *")

		assert.NoError(t, err)
		assert.Equal(t, models.Cron("* * * * *"), c, "cron must be equal to '* * * * *'")
	})

	t.Run("valid cron with minutes", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("5 * * * *")

		assert.NoError(t, err)
		assert.Equal(t, models.Cron("5 * * * *"), c, "cron must be equal to '5 * * * *'")
	})

	t.Run("valid cron with min and hours", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("5 1 * * *")

		assert.NoError(t, err)
		assert.Equal(t, models.Cron("5 1 * * *"), c, "cron must be equal to '5 1 * * *'")
	})

	t.Run("valid cron with /", func(t *testing.T) {
		t.Parallel()
		assert.New(t)

		c, err := models.NewCron("5 0/10 * * *")

		assert.NoError(t, err)
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

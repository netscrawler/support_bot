package service_test

import (
	"support_bot/internal/domain/service"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPassword(t *testing.T) {
	t.Parallel()

	t.Run("valid password", func(t *testing.T) {
		t.Parallel()

		testPassword := "testPassword"

		hash, err := service.Hash(testPassword)
		t.Log(hash)
		require.NoError(t, err)
		require.NotEmpty(t, hash)

		verified, err := service.Verify(testPassword, hash)
		require.NoError(t, err)
		require.True(t, verified)
	})

	t.Run("invalid password", func(t *testing.T) {
		t.Parallel()

		testPassword := "testPassword"

		hash, err := service.Hash(testPassword)
		require.NoError(t, err)
		require.NotEmpty(t, hash)

		verified, err := service.Verify(testPassword+"1", hash)
		require.NoError(t, err)
		require.True(t, !verified)
	})

	t.Run("identical password", func(t *testing.T) {
		t.Parallel()

		testPassword1 := "testPassword1"
		testPassword2 := "testPassword2"

		hash1, err := service.Hash(testPassword1)
		require.NoError(t, err)
		hash2, err := service.Hash(testPassword2)
		require.NoError(t, err)
		require.NotEqual(t, hash1, hash2)
	})
}

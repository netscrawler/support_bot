package store

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckedInt32(t *testing.T) {
	t.Parallel()

	for _, value := range []int{math.MinInt32, 0, math.MaxInt32} {
		got, err := checkedInt32(value)
		require.NoError(t, err)
		require.Equal(t, int32(value), got)
	}

	if int64(^uint(0)>>1) > math.MaxInt32 {
		_, err := checkedInt32(int(int64(math.MaxInt32) + 1))
		require.Error(t, err)

		_, err = checkedInt32(int(int64(math.MinInt32) - 1))
		require.Error(t, err)
	}
}

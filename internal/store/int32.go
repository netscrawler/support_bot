package store

import (
	"fmt"
	"math"
)

func checkedInt32(value int) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("value %d is outside int32 range", value)
	}

	return int32(value), nil
}

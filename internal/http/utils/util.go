package httputils

import (
	"encoding/json"
	"fmt"

	"support_bot/internal/http/errorz"
)

func UnmarshalFor[V any](data []byte) (V, error) {
	var v V
	err := json.Unmarshal(data, &v)
	if err != nil {
		return v, fmt.Errorf("%w: %w", errorz.ErrUnmarshallError, err)
	}
	return v, nil
}

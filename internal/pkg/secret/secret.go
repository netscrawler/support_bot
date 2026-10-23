// Package secret маскирует пароли и токены в логах, JSON и тексте ошибок.
package secret

import (
	"log/slog"
	"slices"
	"strings"
	"sync"
)

const (
	mask = "***"
	// minLen: более короткие значения не регистрируются, чтобы не испортить весь текст.
	minLen = 4
)

// String — секрет; все способы печати дают "***", реальное значение — через Value.
// MarshalYAML намеренно нет: config update должен писать реальное значение.
type String string

func (s String) Value() string      { return string(s) }
func (String) String() string       { return mask }
func (String) GoString() string     { return mask }
func (String) LogValue() slog.Value { return slog.StringValue(mask) }

func (String) MarshalJSON() ([]byte, error) { return []byte(`"` + mask + `"`), nil }

var (
	mu      sync.RWMutex
	secrets []string // по убыванию длины
)

// Register добавляет значения в список для Redact.
func Register(values ...String) {
	mu.Lock()
	defer mu.Unlock()

	for _, v := range values {
		if len(v) < minLen || slices.Contains(secrets, string(v)) {
			continue
		}

		secrets = append(secrets, string(v))
	}

	slices.SortStableFunc(
		secrets,
		func(a, b string) int { return len(b) - len(a) },
	)
}

// Redact заменяет зарегистрированные секреты на "***", длинные первыми.
func Redact(s string) string {
	mu.RLock()
	defer mu.RUnlock()

	for _, v := range secrets {
		s = strings.ReplaceAll(s, v, mask)
	}

	return s
}

func reset() {
	mu.Lock()

	secrets = nil
	mu.Unlock()
}

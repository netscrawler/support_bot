package funcs

import (
	"fmt"
	"strings"
)

func Flatten(m map[string]any, stripPrefixes ...string) map[string]any {
	out := make(map[string]any)

	for k, v := range m {
		flatten(k, v, out, stripPrefixes)
	}

	return out
}

func flatten(prefix string, v any, out map[string]any, stripPrefixes []string) {
	prefix = stripPrefix(prefix, stripPrefixes)

	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			key := k

			if prefix != "" {
				key = prefix + "." + k
			}

			flatten(key, val, out, stripPrefixes)
		}

	case []any:
		// Если массив объектов — разворачиваем по индексам.
		allObjects := true

		for _, item := range x {
			if _, ok := item.(map[string]any); !ok {
				allObjects = false

				break
			}
		}

		if allObjects {
			for i, item := range x {
				flatten(fmt.Sprintf("%s.%d", prefix, i), item, out, stripPrefixes)
			}

			return
		}

		// Остальные массивы оставляем как есть.
		out[prefix] = x

	default:
		out[prefix] = x
	}
}

func stripPrefix(key string, prefixes []string) string {
	for _, p := range prefixes {
		if key == p {
			return ""
		}

		if after, ok := strings.CutPrefix(key, p+"."); ok {
			return after
		}
	}

	return key
}

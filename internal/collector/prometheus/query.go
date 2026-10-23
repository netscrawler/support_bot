package prometheus

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var placeholder = regexp.MustCompile(`\$\{(\w+)\}`)

// buildRequest подставляет ${name} в PromQL и выбирает instant/range по params.
func buildRequest(
	query string,
	params map[string]string,
) (string, url.Values, error) {
	var missing string

	q := placeholder.ReplaceAllStringFunc(query, func(m string) string {
		name := placeholder.FindStringSubmatch(m)[1]

		v, ok := params[name]
		if !ok && missing == "" {
			missing = name
		}

		return v
	})

	if missing != "" {
		return "", nil, fmt.Errorf("в params нет значения для ${%s}", missing)
	}

	vals := url.Values{"query": {q}}

	var absent []string

	for _, k := range []string{"start", "end", "step"} {
		if v, ok := params[k]; ok {
			vals.Set(k, v)
		} else {
			absent = append(absent, k)
		}
	}

	switch len(absent) {
	case 0:
		return "/api/v1/query_range", vals, nil
	case 3:
		if t, ok := params["time"]; ok {
			vals.Set("time", t)
		}

		return "/api/v1/query", vals, nil

	default:
		return "", nil, fmt.Errorf(
			"неполный range-запрос, не хватает: %s",
			strings.Join(absent, ", "),
		)
	}
}

package html

import (
	"fmt"
	"strings"
)

type seriesDefinition struct {
	Field string
	Label string
}

// parseSeries parses "field" or "field:Label" series definitions. A label may
// itself contain a colon.
func parseSeries(values []string) ([]seriesDefinition, error) {
	result := make([]seriesDefinition, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("chart: empty series definition")
		}

		parts := strings.SplitN(value, ":", 2)

		field := strings.TrimSpace(parts[0])
		if field == "" {
			return nil, fmt.Errorf("chart: empty series field in %q", value)
		}

		label := field
		if len(parts) == 2 {
			label = strings.TrimSpace(parts[1])
			if label == "" {
				return nil, fmt.Errorf("chart: empty series label in %q", value)
			}
		}

		result = append(result, seriesDefinition{Field: field, Label: label})
	}

	return result, nil
}

// validateFields checks the x field and every series field appears in at least
// one row. An empty dataset passes (nothing to validate).
func validateFields(rows []map[string]any, xField string, series []seriesDefinition) error {
	if len(rows) == 0 {
		return nil
	}

	xExists := false
	seriesExists := make([]bool, len(series))

	for _, row := range rows {
		if _, ok := row[xField]; ok {
			xExists = true
		}

		for i, s := range series {
			if _, ok := row[s.Field]; ok {
				seriesExists[i] = true
			}
		}
	}

	if !xExists {
		return fmt.Errorf("chart: x field %q not found in dataset", xField)
	}

	for i, exists := range seriesExists {
		if !exists {
			return fmt.Errorf("chart: series field %q not found in dataset", series[i].Field)
		}
	}

	return nil
}

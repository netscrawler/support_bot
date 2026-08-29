package html

import (
	"fmt"
	"html/template"
	"strings"
)

// table-specific heuristics
const (
	rowsPerGridRow  = 6
	headerRowsCount = 1
)

// tableOptions accepts key/value pairs similar to chartOptions.
func tableOptions(args ...any) (map[string]any, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("tableOptions: expected key/value pairs")
	}

	options := make(map[string]any, len(args)/2)

	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			return nil, fmt.Errorf("tableOptions: key at position %d must be string", i)
		}

		options[key] = args[i+1]
	}

	return options, nil
}

// table creates a ReportBlock for a table.
// rows are dataset rows, columns are "field" or "field:Title" definitions.
func (s *gridState) table(
	rows []map[string]any,
	columns []string,
	options map[string]any,
) (string, error) {
	width := 12
	if v, ok := options["w"].(int); ok {
		width = v
	}

	if width < 1 || width > GridColumns {
		return "", fmt.Errorf("table: option \"w\" must be between 1 and %d", GridColumns)
	}

	if len(columns) == 0 {
		return "", fmt.Errorf("table: at least one column is required")
	}

	cols, err := parseColumns(columns)
	if err != nil {
		return "", err
	}

	if err := validateTableFields(rows, cols); err != nil {
		return "", err
	}

	headers := make([]string, 0, len(cols))
	for _, col := range cols {
		headers = append(headers, col.Title)
	}

	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := make([]string, 0, len(cols))

		for _, col := range cols {
			cells = append(cells, fmt.Sprint(row[col.Field]))
		}

		tableRows = append(tableRows, cells)
	}

	h := 0
	explicitH := false
	if v, ok := options["h"].(int); ok {
		h = v
		explicitH = true
	}

	// compute auto height if not explicit
	if !explicitH {
		// header + rows mapped to grid rows
		totalDataRows := len(tableRows)
		// each grid row can contain rowsPerGridRow data rows
		h = headerRowsCount + (totalDataRows+rowsPerGridRow-1)/rowsPerGridRow
		h = max(h, 1)
	}

	// Do not pre-render full content here. Store table data for layout engine to split.
	block := ReportBlock{
		Type:           "table",
		Content:        "",
		Size:           BlockSize{W: width, H: h},
		ExplicitWidth:  true,
		ExplicitHeight: explicitH,
		Splittable:     true,
		TableHeaders:   headers,
		TableRows:      tableRows,
	}

	// If collector active - append and render nothing now
	if len(s.stack) > 0 {
		s.stack[len(s.stack)-1] = append(s.stack[len(s.stack)-1], block)

		return "", nil
	}

	// fallback: render single block immediately using full table content
	content := renderTableChunk(headers, tableRows)
	style := fmt.Sprintf("grid-column: span %d;", width)
	page := fmt.Sprintf(`<div class="report-grid-item" style="%s">%s</div>`, style, content)

	return page, nil
}

type columnDefinition struct {
	Field string
	Title string
}

// parseColumns parses "field" or "field:Title" column definitions.
func parseColumns(values []string) ([]columnDefinition, error) {
	result := make([]columnDefinition, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)

		if value == "" {
			return nil, fmt.Errorf("table: empty column definition")
		}

		parts := strings.SplitN(value, ":", 2)

		field := strings.TrimSpace(parts[0])
		if field == "" {
			return nil, fmt.Errorf("table: empty column field in %q", value)
		}

		title := field

		if len(parts) == 2 {
			title = strings.TrimSpace(parts[1])

			if title == "" {
				return nil, fmt.Errorf("table: empty column title in %q", value)
			}
		}

		result = append(result, columnDefinition{
			Field: field,
			Title: title,
		})
	}

	return result, nil
}

func validateTableFields(rows []map[string]any, cols []columnDefinition) error {
	if len(rows) == 0 {
		return nil
	}

	exists := make([]bool, len(cols))

	for _, row := range rows {
		for i, col := range cols {
			if _, ok := row[col.Field]; ok {
				exists[i] = true
			}
		}
	}

	for i, ok := range exists {
		if !ok {
			return fmt.Errorf("table: column field %q not found in dataset", cols[i].Field)
		}
	}

	return nil
}

// renderTableChunk generates HTML for a table given headers and a subset of rows.
func renderTableChunk(headers []string, rows [][]string) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"chart-card\">\n")
	sb.WriteString("<table style=\"width:100%;border-collapse:collapse;\">\n")
	sb.WriteString("<thead><tr>")

	for _, hname := range headers {
		sb.WriteString(
			fmt.Sprintf(
				"<th style=\"text-align:left;padding:6px;border-bottom:1px solid #e5e9f0;\">%s</th>",
				template.HTMLEscapeString(hname),
			),
		)
	}

	sb.WriteString("</tr></thead>\n")

	sb.WriteString("<tbody>\n")

	for _, row := range rows {
		sb.WriteString("<tr>")

		for _, cell := range row {
			sb.WriteString(
				fmt.Sprintf(
					"<td style=\"padding:6px;border-bottom:1px solid #f3f4f6;\">%s</td>",
					template.HTMLEscapeString(cell),
				),
			)
		}

		sb.WriteString("</tr>\n")
	}

	sb.WriteString("</tbody>\n")
	sb.WriteString("</table>\n")
	sb.WriteString("</div>\n")

	return sb.String()
}

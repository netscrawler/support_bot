package html

import (
	"fmt"
	"html/template"
	"strings"

	"support_bot/internal/models"

	layoutpkg "support_bot/internal/layout"
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

// table is the {{ table }} template helper. Inside an active {{ reportGrid }}
// the block is collected; otherwise a standalone grid item is returned.
// rows are dataset rows, columns are "field" or "field:Title" definitions.
func (s *gridState) table(
	rows []map[string]any,
	columns []string,
	options map[string]any,
) (template.HTML, error) {
	block, err := buildTableModelBlock(columns, options)
	if err != nil {
		return "", err
	}

	if err := validateTableFields(rows, block.Table.Columns); err != nil {
		return "", err
	}

	rb := layoutpkg.ResolvedBlockFromRows(block, rows)

	if len(s.stack) > 0 {
		s.stack[len(s.stack)-1] = append(s.stack[len(s.stack)-1], rb)

		return "", nil
	}

	return standaloneGridItem(rb, s.renderCtx())
}

func buildTableModelBlock(columns []string, options map[string]any) (models.Block, error) {
	if len(columns) == 0 {
		return models.Block{}, fmt.Errorf("table: at least one column is required")
	}

	cols, err := parseColumns(columns)
	if err != nil {
		return models.Block{}, err
	}

	width := layoutpkg.GridColumns
	if v, ok := options["w"].(int); ok {
		width = v
	}

	if width < 1 || width > layoutpkg.GridColumns {
		return models.Block{}, fmt.Errorf(
			"table: option %q must be between 1 and %d", "w", layoutpkg.GridColumns,
		)
	}

	height := 0
	if v, ok := options["h"].(int); ok {
		height = v
	}

	return models.Block{
		ID:    "grid-table",
		Type:  models.BlockTypeTable,
		Pos:   models.GridPosition{W: width, H: height},
		Table: &models.TableBlock{Columns: cols},
	}, nil
}

// parseColumns parses "field" or "field:Title" column definitions.
func parseColumns(values []string) ([]models.Column, error) {
	result := make([]models.Column, 0, len(values))

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

		result = append(result, models.Column{Field: field, Title: title})
	}

	return result, nil
}

func validateTableFields(rows []map[string]any, cols []models.Column) error {
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

	sb.WriteString("<div class=\"table-card\">\n")
	sb.WriteString("<table>\n")
	sb.WriteString("<thead><tr>")

	for _, hname := range headers {
		fmt.Fprintf(
			&sb,
			"<th style=\"text-align:left;padding:6px;border-bottom:1px solid #e5e9f0;\">%s</th>",
			template.HTMLEscapeString(hname),
		)
	}

	sb.WriteString("</tr></thead>\n")
	sb.WriteString("<tbody>\n")

	for _, row := range rows {
		sb.WriteString("<tr>")

		for _, cell := range row {
			fmt.Fprintf(
				&sb,
				"<td style=\"padding:6px;border-bottom:1px solid #f3f4f6;\">%s</td>",
				template.HTMLEscapeString(cell),
			)
		}

		sb.WriteString("</tr>\n")
	}

	sb.WriteString("</tbody>\n")
	sb.WriteString("</table>\n")
	sb.WriteString("</div>\n")

	return sb.String()
}

package layout

import (
	"fmt"
	"sort"
	"strings"

	"support_bot/internal/models"
)

func LayoutFromOrder(order map[string][]string) models.Layout {
	layout := models.Layout{
		Version: 1,
		Page: models.PageConfig{
			Format:      DefaultPageFormat,
			Orientation: DefaultPageOrientation,
			PaddingMM:   DefaultPaddingMM,
		},
	}

	if len(order) == 0 {
		return layout
	}

	keys := make([]string, 0, len(order))
	for key := range order {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	blocks := make([]models.Block, 0, len(keys))
	for _, key := range keys {
		columns := order[key]
		blocks = append(blocks, models.Block{
			ID:      blockID(key, "table"),
			Type:    models.BlockTypeTable,
			Dataset: key,
			Pos: models.GridPosition{
				X: 0,
				Y: 0,
				W: GridColumns,
				H: 0,
			},
			Table: &models.TableBlock{
				Columns: makeColumns(columns),
			},
		})
	}

	layout.Blocks = blocks

	return Normalize(layout)
}

func LayoutFromTemplate(tmpl *models.Template) models.Layout {
	layout := models.Layout{
		Version: 1,
		Page: models.PageConfig{
			Format:      DefaultPageFormat,
			Orientation: DefaultPageOrientation,
			PaddingMM:   DefaultPaddingMM,
		},
	}

	if tmpl == nil {
		return layout
	}

	layout.Blocks = []models.Block{
		{
			ID:   blockID(tmpl.Title, "text"),
			Type: models.BlockTypeText,
			Pos: models.GridPosition{
				X: 0,
				Y: 0,
				W: GridColumns,
				H: 0,
			},
			Text: &models.TextBlock{
				Text: tmpl.TemplateText,
			},
		},
	}

	return Normalize(layout)
}

func makeColumns(definitions []string) []models.Column {
	cols := make([]models.Column, 0, len(definitions))
	for _, definition := range definitions {
		field, title := splitFieldTitle(definition)
		if field == "" {
			continue
		}

		cols = append(cols, models.Column{
			Field: field,
			Title: title,
		})
	}

	return cols
}

func splitFieldTitle(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	field, title, found := strings.Cut(value, ":")
	field = strings.TrimSpace(field)
	if field == "" {
		return "", ""
	}

	if !found {
		return field, field
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return field, field
	}

	return field, title
}

func blockID(parts ...string) string {
	joined := strings.Join(parts, "-")
	var b strings.Builder
	b.Grow(len(joined))

	lastDash := false
	for _, r := range strings.ToLower(joined) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "block"
	}

	return result
}

func LayoutFromExport(export models.Export) models.Layout {
	if export.Layout != nil {
		return Normalize(*export.Layout)
	}

	if export.Template != nil {
		return LayoutFromTemplate(export.Template)
	}

	return LayoutFromOrder(export.Order)
}

func DatasetKeysFromLayout(layout models.Layout) []string {
	keys := make([]string, 0, len(layout.Blocks))
	seen := make(map[string]struct{}, len(layout.Blocks))
	for _, block := range layout.Blocks {
		if block.Dataset == "" {
			continue
		}

		if _, ok := seen[block.Dataset]; ok {
			continue
		}

		seen[block.Dataset] = struct{}{}
		keys = append(keys, block.Dataset)
	}

	sort.Strings(keys)

	return keys
}

func MustValidate(layout models.Layout, datasetKeys []string) {
	if err := Validate(layout, datasetKeys); err != nil {
		panic(fmt.Sprintf("layout validation failed: %v", err))
	}
}

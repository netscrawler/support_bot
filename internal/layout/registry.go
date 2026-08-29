package layout

import (
	"fmt"
	"sort"
	"strings"

	"support_bot/internal/models"
)

const (
	GridColumns = 12
	GridRows    = 12
	DefaultPageFormat      = "A4"
	DefaultPageOrientation = "portrait"
	DefaultPaddingMM       = 16
)

type Definition struct {
	DefaultSize    models.GridPosition
	AllowAutoHeight bool
	Validate        func(models.Block) error
}

var registry = map[models.BlockType]Definition{
	models.BlockTypeTable: {
		DefaultSize:    models.GridPosition{W: GridColumns},
		AllowAutoHeight: true,
		Validate:       validateTableBlock,
	},
	models.BlockTypeChart: {
		DefaultSize:    models.GridPosition{W: 6, H: 4},
		Validate:       validateChartBlock,
	},
	models.BlockTypeMetric: {
		DefaultSize:    models.GridPosition{W: 3, H: 2},
		Validate:       validateMetricBlock,
	},
	models.BlockTypeText: {
		DefaultSize:    models.GridPosition{W: GridColumns},
		AllowAutoHeight: true,
		Validate:       validateTextBlock,
	},
	models.BlockTypeRawHTML: {
		DefaultSize:    models.GridPosition{W: GridColumns},
		AllowAutoHeight: true,
		Validate:       validateRawHTMLBlock,
	},
}

func Definitions() map[models.BlockType]Definition {
	out := make(map[models.BlockType]Definition, len(registry))
	for key, def := range registry {
		out[key] = def
	}

	return out
}

func Validate(layout models.Layout, datasetKeys []string) error {
	normalized := Normalize(layout)
	datasetSet := make(map[string]struct{}, len(datasetKeys))
	for _, key := range datasetKeys {
		if strings.TrimSpace(key) == "" {
			continue
		}

		datasetSet[key] = struct{}{}
	}

	ids := make(map[string]struct{}, len(normalized.Blocks))
	for i := range normalized.Blocks {
		block := normalized.Blocks[i]
		if strings.TrimSpace(block.ID) == "" {
			return fmt.Errorf("block %d: id must not be empty", i)
		}

		if _, ok := ids[block.ID]; ok {
			return fmt.Errorf("block %q: duplicate id", block.ID)
		}
		ids[block.ID] = struct{}{}

		def, ok := registry[block.Type]
		if !ok {
			return fmt.Errorf("block %q: unknown type %q", block.ID, block.Type)
		}

		if err := validateGeometry(block, def); err != nil {
			return fmt.Errorf("block %q: %w", block.ID, err)
		}

		if block.Dataset != "" {
			if _, ok := datasetSet[block.Dataset]; !ok {
				return fmt.Errorf("block %q: dataset %q not found", block.ID, block.Dataset)
			}
		}

		if def.Validate != nil {
			if err := def.Validate(block); err != nil {
				return fmt.Errorf("block %q: %w", block.ID, err)
			}
		}
	}

	return nil
}

func Normalize(layout models.Layout) models.Layout {
	layoutCopy := layout
	layoutCopy.Blocks = make([]models.Block, len(layout.Blocks))
	for i := range layout.Blocks {
		layoutCopy.Blocks[i] = cloneBlock(layout.Blocks[i])
	}

	if layoutCopy.Version <= 0 {
		layoutCopy.Version = 1
	}

	if layoutCopy.Page.Format == "" {
		layoutCopy.Page.Format = DefaultPageFormat
	}

	if layoutCopy.Page.Orientation == "" {
		layoutCopy.Page.Orientation = DefaultPageOrientation
	}

	if layoutCopy.Page.PaddingMM <= 0 {
		layoutCopy.Page.PaddingMM = DefaultPaddingMM
	}

	for i := range layoutCopy.Blocks {
		normalizeBlock(&layoutCopy.Blocks[i])
	}

	return layoutCopy
}

func cloneBlock(block models.Block) models.Block {
	clone := block

	if block.Table != nil {
		table := *block.Table
		table.Columns = append([]models.Column(nil), block.Table.Columns...)
		table.Sort = append([]models.TableSort(nil), block.Table.Sort...)
		clone.Table = &table
	}

	if block.Chart != nil {
		chart := *block.Chart
		chart.Series = append([]models.Series(nil), block.Chart.Series...)
		clone.Chart = &chart
	}

	if block.Metric != nil {
		metric := *block.Metric
		clone.Metric = &metric
	}

	if block.Text != nil {
		text := *block.Text
		clone.Text = &text
	}

	if block.RawHTML != nil {
		rawHTML := *block.RawHTML
		clone.RawHTML = &rawHTML
	}

	return clone
}

func normalizeBlock(block *models.Block) {
	def, ok := registry[block.Type]
	if !ok {
		return
	}

	if block.Pos.W <= 0 {
		block.Pos.W = def.DefaultSize.W
	}

	if block.Pos.H <= 0 && !def.AllowAutoHeight {
		block.Pos.H = def.DefaultSize.H
	}

	switch block.Type {
	case models.BlockTypeTable:
		if block.Table == nil {
			return
		}

		for i := range block.Table.Columns {
			if block.Table.Columns[i].Title == "" {
				block.Table.Columns[i].Title = block.Table.Columns[i].Field
			}
		}
	case models.BlockTypeChart:
		if block.Chart == nil {
			return
		}

		for i := range block.Chart.Series {
			if block.Chart.Series[i].Title == "" {
				block.Chart.Series[i].Title = block.Chart.Series[i].Field
			}
		}
	case models.BlockTypeMetric:
		if block.Metric == nil {
			return
		}

		if block.Metric.Label == "" {
			block.Metric.Label = block.Metric.Field
		}
	}
}

func validateGeometry(block models.Block, def Definition) error {
	if block.Pos.X < 0 || block.Pos.Y < 0 {
		return fmt.Errorf("position must not be negative")
	}

	if block.Pos.W < 1 {
		return fmt.Errorf("option \"w\" must be between 1 and %d", GridColumns)
	}

	if block.Pos.W > GridColumns {
		return fmt.Errorf("option \"w\" must be between 1 and %d", GridColumns)
	}

	if block.Pos.H < 0 {
		return fmt.Errorf("option \"h\" must be greater than or equal to zero")
	}

	if block.Pos.X+block.Pos.W > GridColumns {
		return fmt.Errorf("option \"w\" exceeds grid width")
	}

	if block.Pos.H > 0 && block.Pos.Y+block.Pos.H > GridRows {
		return fmt.Errorf("option \"h\" exceeds grid height")
	}

	if block.Pos.H == 0 && !def.AllowAutoHeight {
		return fmt.Errorf("option \"h\" must be between 1 and %d", GridRows)
	}

	return nil
}

func validateTableBlock(block models.Block) error {
	if block.Table == nil {
		return fmt.Errorf("table configuration is required")
	}

	if strings.TrimSpace(block.Dataset) == "" {
		return fmt.Errorf("table dataset must not be empty")
	}

	if len(block.Table.Columns) == 0 {
		return fmt.Errorf("table must have at least one column")
	}

	seen := make(map[string]struct{}, len(block.Table.Columns))
	for _, column := range block.Table.Columns {
		field := strings.TrimSpace(column.Field)
		if field == "" {
			return fmt.Errorf("table column field must not be empty")
		}

		if _, ok := seen[field]; ok {
			return fmt.Errorf("duplicate table column field %q", field)
		}

		if column.Width < 0 {
			return fmt.Errorf("table column width must not be negative")
		}

		switch column.Align {
		case "", models.ColumnAlignLeft, models.ColumnAlignCenter, models.ColumnAlignRight:
		default:
			return fmt.Errorf("unsupported table column align %q", column.Align)
		}

		seen[field] = struct{}{}
	}

	for _, sortItem := range block.Table.Sort {
		if strings.TrimSpace(sortItem.Field) == "" {
			return fmt.Errorf("table sort field must not be empty")
		}
	}

	if block.Table.Limit != nil && *block.Table.Limit < 0 {
		return fmt.Errorf("table limit must not be negative")
	}

	return nil
}

func validateChartBlock(block models.Block) error {
	if block.Chart == nil {
		return fmt.Errorf("chart configuration is required")
	}

	switch block.Chart.Kind {
	case models.ChartKindLine, models.ChartKindBar, models.ChartKindPie:
	default:
		return fmt.Errorf("unsupported chart kind %q", block.Chart.Kind)
	}

	if strings.TrimSpace(block.Chart.XField) == "" {
		return fmt.Errorf("chart x_field must not be empty")
	}

	if strings.TrimSpace(block.Dataset) == "" {
		return fmt.Errorf("chart dataset must not be empty")
	}

	if len(block.Chart.Series) == 0 {
		return fmt.Errorf("chart must have at least one series")
	}

	seen := make(map[string]struct{}, len(block.Chart.Series))
	for _, series := range block.Chart.Series {
		field := strings.TrimSpace(series.Field)
		if field == "" {
			return fmt.Errorf("chart series field must not be empty")
		}

		if _, ok := seen[field]; ok {
			return fmt.Errorf("duplicate chart series field %q", field)
		}

		seen[field] = struct{}{}
	}

	if block.Chart.Kind == models.ChartKindPie && len(block.Chart.Series) != 1 {
		return fmt.Errorf("pie chart supports exactly one series")
	}

	if block.Chart.Kind == models.ChartKindPie && block.Chart.Stacked {
		return fmt.Errorf("pie chart does not support stacking")
	}

	return nil
}

func validateMetricBlock(block models.Block) error {
	if block.Metric == nil {
		return fmt.Errorf("metric configuration is required")
	}

	if strings.TrimSpace(block.Metric.Field) == "" {
		return fmt.Errorf("metric field must not be empty")
	}

	if strings.TrimSpace(block.Dataset) == "" {
		return fmt.Errorf("metric dataset must not be empty")
	}

	return nil
}

func validateTextBlock(block models.Block) error {
	if block.Text == nil {
		return fmt.Errorf("text configuration is required")
	}

	if strings.TrimSpace(block.Text.Text) == "" {
		return fmt.Errorf("text content must not be empty")
	}

	return nil
}

func validateRawHTMLBlock(block models.Block) error {
	if block.RawHTML == nil {
		return fmt.Errorf("raw_html configuration is required")
	}

	if strings.TrimSpace(block.RawHTML.HTML) == "" {
		return fmt.Errorf("raw_html content must not be empty")
	}

	return nil
}

func SortedDatasetKeys(data map[string][]map[string]any) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

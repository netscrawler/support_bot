package html

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"

	"support_bot/assets"
	"support_bot/internal/exporter/render"
	"support_bot/internal/models"

	layoutpkg "support_bot/internal/layout"
)

type chartType string

const (
	chartTypeLine chartType = "line"
	chartTypeBar  chartType = "bar"
	chartTypePie  chartType = "pie"
)

// escapeHatchChartW is the default grid width for a chart authored via the
// {{ chart }} template helper when "w" is not given. The JSON-layout path uses
// the block registry default instead.
const escapeHatchChartW = 12

// escapeHatchChartH is the default grid height for a chart authored via the
// {{ chart }} template helper when "h" is not given.
const escapeHatchChartH = 4

type chartConfig struct {
	Type    chartType      `json:"type"`
	Data    chartData      `json:"data"`
	Options map[string]any `json:"options,omitempty"`
}

type chartData struct {
	Labels   []any          `json:"labels"`
	Datasets []chartDataset `json:"datasets"`
}

type chartDataset struct {
	Label string `json:"label"`
	Data  []any  `json:"data"`
}

var chartFuncMap = map[string]any{
	"chartOptions": chartOptions,
	"chartJS":      chartJS,
	"chartPalette": chartPalette,
	"styles":       styles,
}

// chart is the {{ chart }} template helper. Inside an active {{ reportGrid }}
// the block is collected for the layout engine and nothing is emitted;
// otherwise a standalone grid item is returned.
func (s *gridState) chart(
	chType string,
	rows []map[string]any,
	xField string,
	series []string,
	options map[string]any,
) (htmltemplate.HTML, error) {
	block, err := buildChartModelBlock(chType, rows, xField, series, options)
	if err != nil {
		return "", err
	}

	rb := layoutpkg.ResolvedBlockFromRows(block, rows)

	if len(s.stack) > 0 {
		s.stack[len(s.stack)-1] = append(s.stack[len(s.stack)-1], rb)

		return "", nil
	}

	return standaloneGridItem(rb, s.renderCtx())
}

// newChart renders a chart outside a reportGrid (used by tests).
func newChart(
	chType string,
	rows []map[string]any,
	xField string,
	series []string,
	options map[string]any,
) (htmltemplate.HTML, error) {
	return newGridState().chart(chType, rows, xField, series, options)
}

func buildChartModelBlock(
	chType string,
	rows []map[string]any,
	xField string,
	series []string,
	options map[string]any,
) (models.Block, error) {
	kind := chartType(chType)
	if kind != chartTypeLine && kind != chartTypeBar && kind != chartTypePie {
		return models.Block{}, fmt.Errorf("chart: invalid chart type: %s", chType)
	}

	if xField == "" {
		return models.Block{}, fmt.Errorf("chart: x field is empty")
	}

	if len(series) == 0 {
		return models.Block{}, fmt.Errorf("chart: at least one series is required")
	}

	if kind == chartTypePie && len(series) > 1 {
		return models.Block{}, fmt.Errorf("chart: pie chart supports exactly one series")
	}

	parsed, err := parseSeries(series)
	if err != nil {
		return models.Block{}, err
	}

	if err := validateFields(rows, xField, parsed); err != nil {
		return models.Block{}, err
	}

	if err := validateChartOptions(kind, options); err != nil {
		return models.Block{}, err
	}

	width := escapeHatchChartW
	if v, ok := options["w"].(int); ok {
		width = v
	}

	if width < 1 || width > layoutpkg.GridColumns {
		return models.Block{}, fmt.Errorf(
			"chart: option %q must be between 1 and %d", "w", layoutpkg.GridColumns,
		)
	}

	height := escapeHatchChartH
	if v, ok := options["h"].(int); ok {
		height = v
	}

	if height < 1 {
		return models.Block{}, fmt.Errorf("chart: option %q must be >= 1", "h")
	}

	mSeries := make([]models.Series, len(parsed))
	for i, p := range parsed {
		mSeries[i] = models.Series{Field: p.Field, Title: p.Label}
	}

	chart := &models.ChartBlock{
		Kind:   models.ChartKind(kind),
		XField: xField,
		Series: mSeries,
	}

	if title, ok := options["title"].(string); ok {
		chart.Title = title
	}

	if canvasH, ok := options["height"].(int); ok {
		chart.HeightPx = canvasH
	}

	if legend, ok := options["legend"].(bool); ok {
		chart.Legend = &legend
	}

	if stacked, ok := options["stacked"].(bool); ok {
		chart.Stacked = stacked
	}

	return models.Block{
		ID:    "grid-chart",
		Type:  models.BlockTypeChart,
		Pos:   models.GridPosition{W: width, H: height},
		Chart: chart,
	}, nil
}

// standaloneGridItem wraps a single resolved block in a report-grid-item for
// rendering outside a {{ reportGrid }}.
func standaloneGridItem(
	rb layoutpkg.ResolvedBlock,
	rc *render.RenderContext,
) (htmltemplate.HTML, error) {
	pb := layoutpkg.PlacedBlock{
		ResolvedBlock: rb,
		Pos:           rb.PreferredSize,
		RowCount:      len(rb.Rows),
	}

	var inner bytes.Buffer
	if err := renderBlock(&inner, rc, pb); err != nil {
		return "", err
	}

	return htmltemplate.HTML(fmt.Sprintf(
		`<div class="report-grid-item" style="grid-column: span %d;">%s</div>`,
		rb.PreferredSize.W, inner.String(),
	)), nil
}

func escapeHTML(value any) string {
	if value == nil {
		return ""
	}

	return htmltemplate.HTMLEscapeString(fmt.Sprint(value))
}

func mustJSONQuote(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func validateChartOptions(kind chartType, options map[string]any) error {
	for key, value := range options {
		switch key {
		case "title":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("chart: option %q must be string", key)
			}

		case "height":
			height, ok := value.(int)
			if !ok {
				return fmt.Errorf("chart: option %q must be int", key)
			}

			if height <= 0 {
				return fmt.Errorf("chart: option %q must be greater than zero", key)
			}

		case "w":
			width, ok := value.(int)
			if !ok {
				return fmt.Errorf("chart: option %q must be int", key)
			}

			if width < 1 || width > layoutpkg.GridColumns {
				return fmt.Errorf("chart: option %q must be between 1 and 12", key)
			}

		case "x", "y":
			position, ok := value.(int)
			if !ok {
				return fmt.Errorf("chart: option %q must be int", key)
			}

			if position < 0 || position >= layoutpkg.GridColumns {
				return fmt.Errorf("chart: option %q must be between 0 and 11", key)
			}

		case "legend":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("chart: option %q must be bool", key)
			}

		case "stacked":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("chart: option %q must be bool", key)
			}

			if kind == chartTypePie {
				return fmt.Errorf("chart: option %q is not supported for pie chart", key)
			}
		}
	}

	return nil
}

func buildChartJSOptions(options map[string]any) map[string]any {
	result := map[string]any{
		"responsive":          true,
		"maintainAspectRatio": false,
		"animation":           false,
	}

	if title, ok := options["title"].(string); ok && title != "" {
		result["plugins"] = map[string]any{
			"title": map[string]any{
				"display": true,
				"text":    title,
			},
		}
	}

	if legend, ok := options["legend"].(bool); ok {
		plugins, exists := result["plugins"].(map[string]any)
		if !exists {
			plugins = map[string]any{}
			result["plugins"] = plugins
		}

		plugins["legend"] = map[string]any{"display": legend}
	}

	if stacked, ok := options["stacked"].(bool); ok && stacked {
		result["scales"] = map[string]any{
			"x": map[string]any{"stacked": true},
			"y": map[string]any{"stacked": true},
		}
	}

	if colors, ok := options["colors"].([]string); ok {
		result["colors"] = colors
	}

	return result
}

func chartOptions(args ...any) (map[string]any, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("chartOptions: expected key/value pairs")
	}

	options := make(map[string]any, len(args)/2)

	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			return nil, fmt.Errorf("chartOptions: key at position %d must be string", i)
		}

		options[key] = args[i+1]
	}

	return options, nil
}

func chartJS() (htmltemplate.HTML, error) {
	data, err := assets.ChartFS.ReadFile("js/chart.umd.min.js")
	if err != nil {
		return "", fmt.Errorf("read chart.js: %w", err)
	}

	return htmltemplate.HTML("<script>\n" + string(data) + "\n</script>"), nil
}

var chartPalettes = map[string][]string{
	"default": {
		"#3B82F6",
		"#10B981",
		"#F59E0B",
		"#EF4444",
		"#8B5CF6",
		"#06B6D4",
	},

	"corporate": {
		"#0066FF",
		"#00A86B",
		"#FFB000",
		"#E53935",
	},
}

func chartPalette(name string) ([]string, error) {
	colors, ok := chartPalettes[name]
	if !ok {
		return nil, fmt.Errorf("unknown chart palette %q", name)
	}

	return colors, nil
}

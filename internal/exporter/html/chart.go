package html

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"strings"
	"support_bot/assets"
	"support_bot/internal/pkg/funcs"
	"sync/atomic"
)

type chartType string

const (
	chartTypeLine chartType = "line"
	chartTypeBar  chartType = "bar"
	chartTypePie  chartType = "pie"
)

type chartSeries struct {
	Label string
	Data  []any
}

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

var chartID uint64

var chartFuncMap = map[string]any{
	"chartOptions": chartOptions,

	"chartJS":      chartJS,
	"chartPalette": chartPalette,
	"styles":       styles,
}

// chart renders a chart block for templates. Inside an active reportGrid the
// block is collected for the layout engine; otherwise fallback HTML is returned.
func (s *gridState) chart(
	chType string,
	rows []map[string]any,
	xField string,
	series []string,
	options map[string]any,
) (htmltemplate.HTML, error) {
	if chType != string(chartTypeLine) && chType != string(chartTypeBar) &&
		chType != string(chartTypePie) {
		return "", fmt.Errorf("chart: invalid chart type: %s", chType)
	}

	block, fallbackHTML, err := buildChartBlock(chartType(chType), rows, xField, series, options)
	if err != nil {
		return "", err
	}

	// if inside reportGrid collector - append and render nothing now
	if len(s.stack) > 0 {
		s.stack[len(s.stack)-1] = append(s.stack[len(s.stack)-1], block)

		return "", nil
	}

	// fallback: return immediate HTML
	return htmltemplate.HTML(fallbackHTML), nil
}

// newChart renders a chart outside a reportGrid (fallback path).
func newChart(
	chType string,
	rows []map[string]any,
	xField string,
	series []string,
	options map[string]any,
) (htmltemplate.HTML, error) {
	return newGridState().chart(chType, rows, xField, series, options)
}

func buildChartBlock(
	chartType chartType,
	rows []map[string]any,
	xField string,
	series []string,
	options map[string]any,
) (ReportBlock, string, error) {
	var empty ReportBlock

	if xField == "" {
		return empty, "", fmt.Errorf("chart: x field is empty")
	}

	if len(series) == 0 {
		return empty, "", fmt.Errorf("chart: at least one series is required")
	}

	if chartType == chartTypePie && len(series) > 1 {
		return empty, "", fmt.Errorf("chart: pie chart supports exactly one series")
	}

	parsedSeries, err := parseSeries(series)
	if err != nil {
		return empty, "", err
	}

	if err := validateFields(rows, xField, parsedSeries); err != nil {
		return empty, "", err
	}

	if err := validateChartOptions(chartType, options); err != nil {
		return empty, "", err
	}

	labels := make([]any, 0, len(rows))

	chartSerial := make([]chartSeries, 0, len(parsedSeries))
	for _, series := range parsedSeries {
		chartSerial = append(chartSerial, chartSeries{
			Label: series.Label,
			Data:  make([]any, 0, len(rows)),
		})
	}

	for _, row := range rows {
		labels = append(labels, row[xField])

		for i, series := range parsedSeries {
			value, ok := row[series.Field]
			if !ok {
				value = nil
			}

			chartSerial[i].Data = append(
				chartSerial[i].Data,
				value,
			)
		}
	}

	datasets := make([]chartDataset, 0, len(chartSerial))
	for _, series := range chartSerial {
		datasets = append(datasets, chartDataset(series))
	}

	defaultOpts := map[string]any{
		"responsive":          true,
		"maintainAspectRatio": false,
		"animation":           false,
		"colors":              chartPalettes["default"],
	}

	opts := funcs.MapJoin(defaultOpts, options)

	config := chartConfig{
		Type: chartType,
		Data: chartData{
			Labels:   labels,
			Datasets: datasets,
		},
		Options: buildChartJSOptions(chartType, opts),
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return empty, "", fmt.Errorf("chart: marshal config: %w", err)
	}

	id := nextChartID()

	height := 300
	if value, ok := options["height"].(int); ok {
		height = value
	}

	// Ширина блока в 12-колоночной сетке.
	width := 12
	if value, ok := options["w"].(int); ok {
		width = value
	}

	if width < 1 || width > 12 {
		return empty, "", fmt.Errorf("chart: option %q must be between 1 and 12", "w")
	}

	// Height in grid units
	hGrid := 4
	if v, ok := options["h"].(int); ok {
		hGrid = v
	}
	if hGrid < 1 {
		return empty, "", fmt.Errorf("chart: option %q must be >= 1", "h")
	}

	// build inner HTML (without grid wrapper)
	inner := fmt.Sprintf(`
		<div class="chart-card">

		    <h3 class="chart-card-title">
		        %s
		    </h3>

		    <div class="report-chart" style="height:%dpx;">
		        <canvas id="%s"></canvas>
		    </div>

		    <script>
		        (function () {
		            const canvas = document.getElementById(%s)

		            if (!canvas) {
		                throw new Error("chart canvas not found: %s");
		            }

		            new Chart(canvas, %s)
		        })();
		    </script>

		</div>
	`, escapeHTML(options["title"]), height, id, mustJSONQuote(id), id, configJSON)

	block := ReportBlock{
		Type:           "chart",
		Content:        inner,
		Size:           BlockSize{W: width, H: hGrid},
		ExplicitWidth:  true,
		ExplicitHeight: true,
		Splittable:     false,
	}

	// fallback wrapper HTML for immediate rendering outside reportGrid
	fallback := fmt.Sprintf(
		`<div class="report-grid-item" style="%s">%s</div>`,
		buildGridItemStyle(width, 0, false, 0, false),
		inner,
	)

	return block, fallback, nil
}

func buildGridItemStyle(
	width int,
	x int,
	hasX bool,
	y int,
	hasY bool,
) string {
	styles := []string{
		fmt.Sprintf("grid-column: span %d", width),
	}

	if hasX {
		styles = append(
			styles,
			fmt.Sprintf("grid-column-start: %d", x+1),
		)
	}

	if hasY {
		styles = append(
			styles,
			fmt.Sprintf("grid-row-start: %d", y+1),
		)
	}

	return strings.Join(styles, "; ")
}

func escapeHTML(value any) string {
	if value == nil {
		return ""
	}

	return htmltemplate.HTMLEscapeString(
		fmt.Sprint(value),
	)
}

type seriesDefinition struct {
	Field string
	Label string
}

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
			return nil, fmt.Errorf(
				"chart: empty series field in %q",
				value,
			)
		}

		label := field

		if len(parts) == 2 {
			label = strings.TrimSpace(parts[1])

			if label == "" {
				return nil, fmt.Errorf(
					"chart: empty series label in %q",
					value,
				)
			}
		}

		result = append(result, seriesDefinition{
			Field: field,
			Label: label,
		})
	}

	return result, nil
}

func validateFields(
	rows []map[string]any,
	xField string,
	series []seriesDefinition,
) error {
	if len(rows) == 0 {
		return nil
	}

	xExists := false
	seriesExists := make([]bool, len(series))

	for _, row := range rows {
		if _, ok := row[xField]; ok {
			xExists = true
		}

		for i, series := range series {
			if _, ok := row[series.Field]; ok {
				seriesExists[i] = true
			}
		}
	}

	if !xExists {
		return fmt.Errorf(
			"chart: x field %q not found in dataset",
			xField,
		)
	}

	for i, exists := range seriesExists {
		if !exists {
			return fmt.Errorf(
				"chart: series field %q not found in dataset",
				series[i].Field,
			)
		}
	}

	return nil
}

func nextChartID() string {
	id := atomic.AddUint64(&chartID, 1)

	return fmt.Sprintf("report-chart-%d", id)
}

func mustJSONQuote(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func validateChartOptions(
	chartType chartType,
	options map[string]any,
) error {
	for key, value := range options {
		switch key {
		case "title":
			if _, ok := value.(string); !ok {
				return fmt.Errorf(
					"chart: option %q must be string",
					key,
				)
			}

		case "height":
			height, ok := value.(int)
			if !ok {
				return fmt.Errorf(
					"chart: option %q must be int",
					key,
				)
			}

			if height <= 0 {
				return fmt.Errorf(
					"chart: option %q must be greater than zero",
					key,
				)
			}

		case "w":
			width, ok := value.(int)
			if !ok {
				return fmt.Errorf(
					"chart: option %q must be int",
					key,
				)
			}

			if width < 1 || width > 12 {
				return fmt.Errorf(
					"chart: option %q must be between 1 and 12",
					key,
				)
			}

		case "x", "y":
			position, ok := value.(int)
			if !ok {
				return fmt.Errorf(
					"chart: option %q must be int",
					key,
				)
			}

			if position < 0 || position >= 12 {
				return fmt.Errorf(
					"chart: option %q must be between 0 and 11",
					key,
				)
			}

		case "legend":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf(
					"chart: option %q must be bool",
					key,
				)
			}

		case "stacked":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf(
					"chart: option %q must be bool",
					key,
				)
			}

			if chartType == chartTypePie {
				return fmt.Errorf(
					"chart: option %q is not supported for pie chart",
					key,
				)
			}
		}
	}

	return nil
}

func buildChartJSOptions(
	chartType chartType,
	options map[string]any,
) map[string]any {
	result := map[string]any{
		"responsive":          true,
		"maintainAspectRatio": false,
		"animation":           false,
	}

	if title, ok := options["title"].(string); ok && title != "" {
		plugins := map[string]any{
			"title": map[string]any{
				"display": true,
				"text":    title,
			},
		}

		result["plugins"] = plugins
	}

	if legend, ok := options["legend"].(bool); ok {
		plugins, exists := result["plugins"].(map[string]any)
		if !exists {
			plugins = map[string]any{}
			result["plugins"] = plugins
		}

		plugins["legend"] = map[string]any{
			"display": legend,
		}
	}

	if stacked, ok := options["stacked"].(bool); ok && stacked {
		result["scales"] = map[string]any{
			"x": map[string]any{
				"stacked": true,
			},
			"y": map[string]any{
				"stacked": true,
			},
		}
	}
	if colors, ok := options["colors"].([]string); ok {
		result["colors"] = colors
	}

	return result
}

func chartOptions(args ...any) (map[string]any, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf(
			"chartOptions: expected key/value pairs",
		)
	}

	options := make(map[string]any, len(args)/2)

	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			return nil, fmt.Errorf(
				"chartOptions: key at position %d must be string",
				i,
			)
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

	return htmltemplate.HTML(
		"<script>\n" +
			string(data) +
			"\n</script>",
	), nil
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

package html

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io"
	"strings"

	"support_bot/internal/exporter/render"
	layoutpkg "support_bot/internal/layout"
	"support_bot/internal/models"
)

const defaultChartCanvasPx = 300

// blockFn renders one placed block's inner HTML (the content of a
// .report-grid-item) for the pure-layout path.
type blockFn func(w io.Writer, rc *render.RenderContext, pb layoutpkg.PlacedBlock) error

var blocks = map[models.BlockType]blockFn{
	models.BlockTypeChart:   renderChartBlock,
	models.BlockTypeTable:   renderTableBlock,
	models.BlockTypeMetric:  renderMetricBlock,
	models.BlockTypeText:    renderTextBlock,
	models.BlockTypeRawHTML: renderRawHTMLBlock,
}

func renderBlock(w io.Writer, rc *render.RenderContext, pb layoutpkg.PlacedBlock) error {
	fn, ok := blocks[pb.Block.Type]
	if !ok {
		return fmt.Errorf("html: no renderer for block type %q", pb.Block.Type)
	}

	return fn(w, rc, pb)
}

const chartCardTemplate = `
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
`

func renderChartBlock(w io.Writer, rc *render.RenderContext, pb layoutpkg.PlacedBlock) error {
	ch := pb.Block.Chart
	if ch == nil {
		return fmt.Errorf("html: chart block %q missing chart config", pb.Block.ID)
	}

	rows := pb.Rows

	labels := make([]any, 0, len(rows))
	datasets := make([]chartDataset, len(ch.Series))
	for i, s := range ch.Series {
		datasets[i] = chartDataset{Label: s.TitleOrField(), Data: make([]any, 0, len(rows))}
	}

	for _, row := range rows {
		labels = append(labels, row[ch.XField])

		for i, s := range ch.Series {
			datasets[i].Data = append(datasets[i].Data, row[s.Field])
		}
	}

	opts := map[string]any{}
	if ch.Stacked {
		opts["stacked"] = true
	}

	if ch.Title != "" {
		opts["title"] = ch.Title
	}

	if ch.Legend != nil {
		opts["legend"] = *ch.Legend
	}

	if colors := seriesColors(ch.Series); len(colors) > 0 {
		opts["colors"] = colors
	} else {
		opts["colors"] = chartPalettes["default"]
	}

	cfg := chartConfig{
		Type:    chartType(ch.Kind),
		Data:    chartData{Labels: labels, Datasets: datasets},
		Options: buildChartJSOptions(opts),
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("html: marshal chart %q: %w", pb.Block.ID, err)
	}

	id := rc.NextID()

	canvasHeight := ch.HeightPx
	if canvasHeight <= 0 {
		canvasHeight = defaultChartCanvasPx
	}

	fmt.Fprintf(
		w, chartCardTemplate,
		escapeHTML(ch.Title), canvasHeight, id, mustJSONQuote(id), id, cfgJSON,
	)

	return nil
}

func seriesColors(series []models.Series) []string {
	out := make([]string, 0, len(series))
	for _, s := range series {
		if strings.TrimSpace(s.Color) == "" {
			return nil
		}

		out = append(out, s.Color)
	}

	return out
}

func renderTableBlock(w io.Writer, _ *render.RenderContext, pb layoutpkg.PlacedBlock) error {
	tb := pb.Block.Table
	if tb == nil {
		return fmt.Errorf("html: table block %q missing table config", pb.Block.ID)
	}

	end := min(pb.RowOffset+pb.RowCount, len(pb.Rows))
	win := pb.Rows[pb.RowOffset:end]

	headers := make([]string, len(tb.Columns))
	for i, c := range tb.Columns {
		headers[i] = c.Title
	}

	cells := make([][]string, 0, len(win))
	for _, row := range win {
		r := make([]string, len(tb.Columns))
		for i, c := range tb.Columns {
			r[i] = fmt.Sprint(row[c.Field])
		}

		cells = append(cells, r)
	}

	_, err := io.WriteString(w, renderTableChunk(headers, cells))

	return err
}

const metricCardTemplate = `<div class="metric-card">` +
	`<div class="metric-label">%s</div>` +
	`<div class="metric-value">%s</div>` +
	`</div>`

func renderMetricBlock(w io.Writer, _ *render.RenderContext, pb layoutpkg.PlacedBlock) error {
	mb := pb.Block.Metric
	if mb == nil {
		return fmt.Errorf("html: metric block %q missing metric config", pb.Block.ID)
	}

	value := ""
	for _, row := range pb.Rows {
		if v, ok := row[mb.Field]; ok {
			value = fmt.Sprint(v)

			break
		}
	}

	label := mb.Label
	if label == "" {
		label = mb.Field
	}

	fmt.Fprintf(
		w, metricCardTemplate,
		htmltemplate.HTMLEscapeString(label), htmltemplate.HTMLEscapeString(value),
	)

	return nil
}

func renderTextBlock(w io.Writer, rc *render.RenderContext, pb layoutpkg.PlacedBlock) error {
	tb := pb.Block.Text
	if tb == nil {
		return fmt.Errorf("html: text block %q missing text config", pb.Block.ID)
	}

	t, err := htmltemplate.New("text_block").Funcs(rc.Funcs).Parse(tb.Text)
	if err != nil {
		return fmt.Errorf("html: parse text block %q: %w", pb.Block.ID, err)
	}

	if _, err := io.WriteString(w, `<div class="report-text">`); err != nil {
		return err
	}

	if err := t.Execute(w, rc.Data); err != nil {
		return fmt.Errorf("html: exec text block %q: %w", pb.Block.ID, err)
	}

	_, err = io.WriteString(w, `</div>`)

	return err
}

func renderRawHTMLBlock(w io.Writer, _ *render.RenderContext, pb layoutpkg.PlacedBlock) error {
	rb := pb.Block.RawHTML
	if rb == nil {
		return fmt.Errorf("html: raw_html block %q missing raw_html config", pb.Block.ID)
	}

	_, err := io.WriteString(w, rb.HTML)

	return err
}

package html

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io"
	"strings"

	"support_bot/internal/exporter/render"
	"support_bot/internal/models"

	layoutpkg "support_bot/internal/layout"
)

// layoutChartPalette is the default categorical palette for declarative-layout
// charts. A series with an explicit color overrides its slot; unset series fill
// from here by index.
var layoutChartPalette = []string{
	"#509EE3", "#88BF4D", "#A989C5", "#EF8C8C",
	"#F9D45C", "#98D9D9", "#7172AD", "#F2A86F",
}

// defaultLayoutChartPx is the canvas height (px) for a chart block that sets no
// height_px. It is a min-height: the card grows to fill a taller row.
const defaultLayoutChartPx = 240

// errWriter accumulates the first write error so the many small writes below do
// not each need an inline check.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) Write(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}

	var n int
	n, e.err = e.w.Write(p)

	return n, e.err
}

func (e *errWriter) str(s string) {
	if e.err == nil {
		_, e.err = io.WriteString(e.w, s)
	}
}

func (e *errWriter) printf(format string, a ...any) {
	if e.err == nil {
		_, e.err = fmt.Fprintf(e.w, format, a...)
	}
}

// renderLayoutDocument writes the full <body> for the declarative-layout path:
// one .report-page per packed page, each block placed on its flow row inside a
// flex .report-grid. Pages carry min-height, not height, so nothing is clipped.
func renderLayoutDocument(
	w io.Writer,
	rc *render.RenderContext,
	pages []layoutpkg.Page,
	page models.PageConfig,
) error {
	pad := page.PaddingMM
	if pad <= 0 {
		pad = layoutpkg.DefaultPaddingMM
	}

	out := &errWriter{w: w}

	for _, p := range pages {
		out.printf(`<div class="report-page" style="padding: %dmm;">`+"\n", pad)

		if err := renderPageSection(out, rc, page.Header, "header", "report-header"); err != nil {
			return err
		}

		out.str(`<div class="report-grid">` + "\n")

		row := -1

		for _, pb := range p.Blocks {
			if pb.Row != row {
				if row != -1 {
					out.str("</div>\n")
				}

				out.str(`<div class="report-row">` + "\n")
				row = pb.Row
			}

			out.printf(`<div class="report-grid-item" style="flex-grow: %d;">`, pb.Pos.W)

			if err := renderLayoutBlock(out, rc, pb); err != nil {
				return err
			}

			out.str("</div>\n")
		}

		if row != -1 {
			out.str("</div>\n")
		}

		out.str("</div>\n")

		if err := renderPageSection(out, rc, page.Footer, "footer", "report-footer"); err != nil {
			return err
		}

		out.str("</div>\n")
	}

	return out.err
}

func renderLayoutBlock(out *errWriter, rc *render.RenderContext, pb layoutpkg.PlacedBlock) error {
	switch pb.Block.Type {
	case models.BlockTypeChart:
		return renderLayoutChartBlock(out, rc, pb)
	case models.BlockTypeTable:
		return renderLayoutTableBlock(out, pb)
	case models.BlockTypeMetric:
		return renderLayoutMetricBlock(out, pb)
	case models.BlockTypeText:
		return renderLayoutTextBlock(out, rc, pb)
	case models.BlockTypeRawHTML:
		return renderLayoutRawHTMLBlock(out, pb)
	default:
		return fmt.Errorf("html: no layout renderer for block type %q", pb.Block.Type)
	}
}

type layoutChartDataset struct {
	Label           string  `json:"label"`
	Data            []any   `json:"data"`
	BackgroundColor any     `json:"backgroundColor,omitempty"`
	BorderColor     any     `json:"borderColor,omitempty"`
	Fill            bool    `json:"fill"`
	Tension         float64 `json:"tension,omitempty"`
}

type layoutChartConfig struct {
	Type    string          `json:"type"`
	Data    layoutChartData `json:"data"`
	Options map[string]any  `json:"options"`
}

type layoutChartData struct {
	Labels   []any                `json:"labels"`
	Datasets []layoutChartDataset `json:"datasets"`
}

func renderLayoutChartBlock(
	out *errWriter,
	rc *render.RenderContext,
	pb layoutpkg.PlacedBlock,
) error {
	ch := pb.Block.Chart
	if ch == nil {
		return fmt.Errorf("html: chart block %q missing chart config", pb.Block.ID)
	}

	rows := pb.Rows
	labels := make([]any, 0, len(rows))

	for _, r := range rows {
		labels = append(labels, r[ch.XField])
	}

	colors := layoutSeriesColors(ch.Series)
	datasets := make([]layoutChartDataset, 0, len(ch.Series))

	for i, s := range ch.Series {
		values := make([]any, 0, len(rows))
		for _, r := range rows {
			values = append(values, r[s.Field])
		}

		ds := layoutChartDataset{Label: s.TitleOrField(), Data: values}

		switch ch.Kind {
		case models.ChartKindPie:
			ds.BackgroundColor = sliceColors(len(rows))
		case models.ChartKindBar:
			ds.BackgroundColor = colors[i]
			ds.BorderColor = colors[i]
		default: // line
			ds.BorderColor = colors[i]
			ds.BackgroundColor = colors[i]
			ds.Tension = 0.3
		}

		datasets = append(datasets, ds)
	}

	cfg := layoutChartConfig{
		Type:    string(chartType(ch.Kind)),
		Data:    layoutChartData{Labels: labels, Datasets: datasets},
		Options: buildLayoutChartOptions(ch),
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("html: marshal chart %q: %w", pb.Block.ID, err)
	}

	id := rc.NextID()

	height := ch.HeightPx
	if height <= 0 {
		height = defaultLayoutChartPx
	}

	out.str(`<div class="card">`)

	if title := strings.TrimSpace(ch.Title); title != "" {
		out.printf(`<h3 class="card-title">%s</h3>`, escapeHTML(title))
	}

	out.printf(`<div class="report-chart" style="min-height: %dpx;">`, height)
	out.printf(`<canvas id="%s"></canvas></div>`, id)
	out.printf(
		`<script>(function(){var c=document.getElementById(%s);`+
			`if(c){new Chart(c,%s);}})();</script>`,
		mustJSONQuote(id), cfgJSON,
	)
	out.str(`</div>`)

	return out.err
}

// layoutSeriesColors resolves one colour per series: the series' own Color when
// set, otherwise the palette entry for its index.
func layoutSeriesColors(series []models.Series) []string {
	out := make([]string, len(series))

	for i, s := range series {
		if c := strings.TrimSpace(s.Color); c != "" {
			out[i] = c

			continue
		}

		out[i] = layoutChartPalette[i%len(layoutChartPalette)]
	}

	return out
}

// sliceColors returns n palette colours, cycled, for the slices of a pie chart.
func sliceColors(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = layoutChartPalette[i%len(layoutChartPalette)]
	}

	return out
}

func buildLayoutChartOptions(ch *models.ChartBlock) map[string]any {
	legend := true
	if ch.Legend != nil {
		legend = *ch.Legend
	}

	opts := map[string]any{
		"responsive":          true,
		"maintainAspectRatio": false,
		"animation":           false,
		"plugins": map[string]any{
			"legend": map[string]any{"display": legend},
		},
	}

	if ch.Kind != models.ChartKindPie {
		yScale := map[string]any{"beginAtZero": true}
		xScale := map[string]any{}

		if ch.Stacked {
			yScale["stacked"] = true
			xScale["stacked"] = true
		}

		opts["scales"] = map[string]any{"x": xScale, "y": yScale}
	}

	return opts
}

func renderLayoutTableBlock(out *errWriter, pb layoutpkg.PlacedBlock) error {
	tb := pb.Block.Table
	if tb == nil {
		return fmt.Errorf("html: table block %q missing table config", pb.Block.ID)
	}

	end := min(pb.RowOffset+pb.RowCount, len(pb.Rows))
	win := pb.Rows[pb.RowOffset:end]

	// overflow: hidden clips only the rounded corners — the card auto-sizes to
	// the table height (rows are never hidden) and every table owns a
	// full-width row so nothing is clipped horizontally.
	out.str(`<div class="card" style="padding: 0; overflow: hidden;">`)
	out.str(`<table class="report-table">`)

	writeColGroup(out, tb.Columns)

	out.str(`<thead><tr>`)

	for _, c := range tb.Columns {
		out.printf(`<th class="%s">%s</th>`, columnClasses(c), escapeHTML(c.Title))
	}

	out.str(`</tr></thead><tbody>`)

	for _, r := range win {
		out.str(`<tr>`)

		for _, c := range tb.Columns {
			out.printf(
				`<td class="%s">%s</td>`,
				columnClasses(c),
				htmltemplate.HTMLEscapeString(layoutpkg.FormatValue(r[c.Field], c.Format)),
			)
		}

		out.str(`</tr>`)
	}

	out.str(`</tbody></table></div>`)

	return out.err
}

// writeColGroup emits a <colgroup> only when at least one column pins an
// explicit width; the tight numeric columns are sized from a cell class
// instead (a <col> cannot carry white-space).
func writeColGroup(out *errWriter, cols []models.Column) {
	pinned := false

	for _, c := range cols {
		if c.Width > 0 {
			pinned = true

			break
		}
	}

	if !pinned {
		return
	}

	out.str(`<colgroup>`)

	for _, c := range cols {
		if c.Width > 0 {
			out.printf(`<col style="width: %dpx">`, c.Width)

			continue
		}

		out.str(`<col>`)
	}

	out.str(`</colgroup>`)
}

// columnClasses is the class list for a column's <th> and <td>: an alignment
// class (explicit Align, else right for a numeric Format, else left) plus
// col-num for an auto-width numeric column so it shrinks to its content and
// the label column keeps the slack.
func columnClasses(c models.Column) string {
	cls := alignClass(c)

	if c.Width == 0 && isNumericFormat(c.Format) {
		cls += " col-num"
	}

	return cls
}

func alignClass(c models.Column) string {
	switch c.Align {
	case models.ColumnAlignRight:
		return "align-right"
	case models.ColumnAlignCenter:
		return "align-center"
	case models.ColumnAlignLeft:
		return "align-left"
	}

	if isNumericFormat(c.Format) {
		return "align-right"
	}

	return "align-left"
}

func isNumericFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "number", "percent", "money":
		return true
	default:
		return false
	}
}

func renderLayoutMetricBlock(out *errWriter, pb layoutpkg.PlacedBlock) error {
	mb := pb.Block.Metric
	if mb == nil {
		return fmt.Errorf("html: metric block %q missing metric config", pb.Block.ID)
	}

	var value any

	for _, r := range pb.Rows {
		if v, ok := r[mb.Field]; ok {
			value = v

			break
		}
	}

	label := mb.Label
	if label == "" {
		label = mb.Field
	}

	out.printf(
		`<div class="card metric-card">`+
			`<div class="metric-label">%s</div>`+
			`<div class="metric-value">%s</div>`+
			`</div>`,
		htmltemplate.HTMLEscapeString(label),
		htmltemplate.HTMLEscapeString(layoutpkg.FormatValue(value, mb.Format)),
	)

	return out.err
}

func renderLayoutTextBlock(
	out *errWriter,
	rc *render.RenderContext,
	pb layoutpkg.PlacedBlock,
) error {
	tb := pb.Block.Text
	if tb == nil {
		return fmt.Errorf("html: text block %q missing text config", pb.Block.ID)
	}

	t, err := htmltemplate.New("layout_text").Funcs(rc.Funcs).Parse(tb.Text)
	if err != nil {
		return fmt.Errorf("html: parse text block %q: %w", pb.Block.ID, err)
	}

	out.str(`<div class="card"><div class="report-text">`)

	if out.err == nil {
		out.err = t.Execute(out, rc.Data)
	}

	out.str(`</div></div>`)

	if out.err != nil {
		return fmt.Errorf("html: text block %q: %w", pb.Block.ID, out.err)
	}

	return nil
}

func renderLayoutRawHTMLBlock(out *errWriter, pb layoutpkg.PlacedBlock) error {
	rb := pb.Block.RawHTML
	if rb == nil {
		return fmt.Errorf("html: raw_html block %q missing raw_html config", pb.Block.ID)
	}

	out.str(rb.HTML)

	return out.err
}

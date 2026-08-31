package xlsx

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	texttemplate "text/template"

	"support_bot/internal/models"

	"github.com/xuri/excelize/v2"

	layoutpkg "support_bot/internal/layout"

	textfuncs "support_bot/internal/pkg/text"
)

// The declarative-layout workbook is a linear document: one sheet, blocks
// stacked top to bottom in declaration order. A grid-coordinate emulation was
// tried and abandoned — a spreadsheet is not a canvas, so absolute placement
// only produced overlapping charts and colliding column widths.
const (
	layoutSheet       = "Отчёт"
	layoutDataSheet   = "_source"
	layoutContentCols = 8 // section content spans columns A..H
	layoutChartRows   = 13
	layoutColWidth    = 13.0
	layoutFirstColW   = 20.0
	xlsxPaperA4       = 9
)

var htmlTagRE = regexp.MustCompile(`<[^>]*>`)

// buildLayoutWorkbook renders a resolved declarative document to a single-sheet
// workbook.
func buildLayoutWorkbook(
	data models.Dataset,
	doc layoutpkg.ResolvedDocument,
) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	if err := f.SetSheetName("Sheet1", layoutSheet); err != nil {
		return nil, fmt.Errorf("set sheet name: %w", err)
	}

	styles, err := newLayoutStyles(f)
	if err != nil {
		return nil, err
	}

	w := &layoutWriter{
		f:       f,
		sheet:   layoutSheet,
		data:    data,
		styles:  styles,
		row:     1,
		maxCol:  1,
		dataRow: 1,
	}

	if s := w.sectionText(doc.Page.Header); s != "" {
		w.writeTitle(s)
	}

	blocks := doc.Blocks
	for i := 0; i < len(blocks); {
		if blocks[i].Block.Type == models.BlockTypeMetric {
			j := i
			for j < len(blocks) && blocks[j].Block.Type == models.BlockTypeMetric {
				j++
			}

			w.writeMetricStrip(blocks[i:j])
			i = j

			continue
		}

		w.writeBlock(blocks[i])
		i++
	}

	if s := w.sectionText(doc.Page.Footer); s != "" {
		w.writeNoteText(s)
	}

	if w.err != nil {
		return nil, w.err
	}

	if w.hasData {
		if err := f.SetSheetVisible(layoutDataSheet, false); err != nil {
			return nil, fmt.Errorf("hide data sheet: %w", err)
		}
	}

	w.finish(doc.Page.Orientation)
	if w.err != nil {
		return nil, w.err
	}

	f.SetActiveSheet(0)
	f.SetAppProps(&excelize.AppProperties{Application: "SendyStats", Company: "Sendy"})

	return f.WriteToBuffer()
}

// layoutWriter walks the blocks with a downward row cursor, accumulating the
// first write error so the many small excelize calls need no inline checks.
type layoutWriter struct {
	f      *excelize.File
	sheet  string
	data   models.Dataset
	styles layoutStyles

	row      int // next free 1-based worksheet row
	maxCol   int // widest column index written, for the final width pass
	tableSeq int

	dataRow int // next free row on the hidden data sheet
	hasData bool

	err error
}

func (w *layoutWriter) touch(col int) {
	if col > w.maxCol {
		w.maxCol = col
	}
}

// set writes one styled cell.
func (w *layoutWriter) set(col, row int, value any, style int) {
	if w.err != nil {
		return
	}

	cell := cellName(col, row)
	if w.err = w.f.SetCellValue(w.sheet, cell, value); w.err != nil {
		return
	}

	w.err = w.f.SetCellStyle(w.sheet, cell, cell, style)
	w.touch(col)
}

// mergeRange writes value into a horizontal merge from startCol to endCol.
func (w *layoutWriter) mergeRange(startCol, endCol, row int, value any, style int) {
	if w.err != nil {
		return
	}

	a, b := cellName(startCol, row), cellName(endCol, row)
	if w.err = w.f.MergeCell(w.sheet, a, b); w.err != nil {
		return
	}

	if w.err = w.f.SetCellValue(w.sheet, a, value); w.err != nil {
		return
	}

	w.err = w.f.SetCellStyle(w.sheet, a, b, style)
	w.touch(endCol)
}

func (w *layoutWriter) rowHeight(row int, h float64) {
	if w.err == nil {
		w.err = w.f.SetRowHeight(w.sheet, row, h)
	}
}

func (w *layoutWriter) writeTitle(text string) {
	w.mergeRange(1, layoutContentCols, w.row, text, w.styles.title)
	w.rowHeight(w.row, 26)
	w.row += 2
}

func (w *layoutWriter) writeHeading(text string) {
	w.mergeRange(1, layoutContentCols, w.row, text, w.styles.heading)
	w.row++
}

func (w *layoutWriter) writeNoteText(text string) {
	w.mergeRange(1, layoutContentCols, w.row, text, w.styles.note)
	w.row += 2
}

func (w *layoutWriter) writeBlock(rb layoutpkg.ResolvedBlock) {
	if w.err != nil {
		return
	}

	switch rb.Block.Type {
	case models.BlockTypeChart:
		w.writeChart(rb)
	case models.BlockTypeTable:
		w.writeTable(rb)
	case models.BlockTypeText:
		w.writeText(rb)
	case models.BlockTypeRawHTML:
		w.writeRawHTML(rb)
	case models.BlockTypeMetric:
		w.writeMetricStrip([]layoutpkg.ResolvedBlock{rb})
	default:
		w.err = fmt.Errorf("no renderer for block type %q", rb.Block.Type)
	}
}

// writeMetricStrip lays a run of metric blocks out as label / value bands, up
// to four per row.
func (w *layoutWriter) writeMetricStrip(metrics []layoutpkg.ResolvedBlock) {
	const perStrip = 4

	for off := 0; off < len(metrics); off += perStrip {
		grp := metrics[off:min(off+perStrip, len(metrics))]
		span := max(layoutContentCols/len(grp), 1)

		labelRow, valueRow := w.row, w.row+1

		for k, rb := range grp {
			mb := rb.Block.Metric
			if mb == nil {
				w.err = fmt.Errorf("metric block %q missing config", rb.Block.ID)

				return
			}

			startCol := k*span + 1
			endCol := startCol + span - 1

			label := mb.Label
			if label == "" {
				label = mb.Field
			}

			// Uppercase mirrors the html .metric-label.
			w.mergeRange(startCol, endCol, labelRow, strings.ToUpper(label), w.styles.kpiLabel)
			w.mergeRange(startCol, endCol, valueRow, metricValue(rb.Rows, mb), w.styles.kpiValue)
		}

		w.rowHeight(valueRow, 28)
		w.row += 3
	}
}

func metricValue(rows []map[string]any, mb *models.MetricBlock) string {
	for _, r := range rows {
		if v, ok := r[mb.Field]; ok {
			return layoutpkg.FormatValue(v, mb.Format)
		}
	}

	return ""
}

func (w *layoutWriter) writeTable(rb layoutpkg.ResolvedBlock) {
	tb := rb.Block.Table
	if tb == nil {
		w.err = fmt.Errorf("table block %q missing config", rb.Block.ID)

		return
	}

	cols := tb.Columns
	headerRow := w.row

	for i, c := range cols {
		title := c.Title
		if title == "" {
			title = c.Field
		}

		w.set(i+1, headerRow, title, w.styles.header[alignIndex(c)])
	}

	for r, row := range rb.Rows {
		for i, c := range cols {
			w.set(
				i+1,
				headerRow+1+r,
				layoutpkg.FormatValue(row[c.Field], c.Format),
				w.styles.body[alignIndex(c)][r%2],
			)
		}
	}

	endRow := headerRow + len(rb.Rows)

	// AddTable supplies the autofilter and a named structured range; its
	// visual style is left at the minimal built-in because the header and
	// zebra are painted by the cell styles above.
	if len(rb.Rows) > 0 && w.err == nil {
		w.tableSeq++
		w.err = w.f.AddTable(w.sheet, &excelize.Table{
			Range:         cellRange(1, headerRow, len(cols), endRow),
			Name:          fmt.Sprintf("tbl%d", w.tableSeq),
			StyleName:     "TableStyleLight1",
			ShowHeaderRow: boolPtr(true),
		})
	}

	w.touch(len(cols))
	w.row = endRow + 2
}

// alignIndex maps a column to a header/body style slot, mirroring the html
// renderer: explicit Align wins, else a numeric Format reads right-aligned,
// else left.
func alignIndex(c models.Column) int {
	if i, ok := alignByName[c.Align]; ok {
		return i
	}

	if isNumericFormat(c.Format) {
		return alignByName[models.ColumnAlignRight]
	}

	return alignByName[models.ColumnAlignLeft]
}

func isNumericFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "number", "percent", "money":
		return true
	default:
		return false
	}
}

func (w *layoutWriter) writeChart(rb layoutpkg.ResolvedBlock) {
	ch := rb.Block.Chart
	if ch == nil {
		w.err = fmt.Errorf("chart block %q missing config", rb.Block.ID)

		return
	}

	if len(ch.Series) == 0 {
		w.err = fmt.Errorf("chart block %q has no series", rb.Block.ID)

		return
	}

	if title := strings.TrimSpace(ch.Title); title != "" {
		w.writeHeading(title)
	}

	sheet := w.ensureDataSheet()
	start := w.dataRow
	n := len(rb.Rows)

	w.writeSeriesData(sheet, start, rb.Rows, ch)
	if w.err != nil {
		return
	}

	cats := chartRef(sheet, 1, start+1, 1, start+n)

	// No chart.Title: the heading row above already labels it, and an in-plot
	// title just overflows the anchored chart box.
	chart := &excelize.Chart{
		Type:       chartType(ch.Kind, ch.Stacked),
		Dimension:  excelize.ChartDimension{Width: 460, Height: 240},
		Legend:     excelize.ChartLegend{Position: legendPosition(ch.Legend)},
		VaryColors: boolPtr(ch.Kind == models.ChartKindPie),
	}

	for i := range ch.Series {
		col := i + 2
		chart.Series = append(chart.Series, excelize.ChartSeries{
			Name:       chartRef(sheet, col, start, col, start),
			Categories: cats,
			Values:     chartRef(sheet, col, start+1, col, start+n),
		})
	}

	if w.err = w.f.AddChart(w.sheet, cellName(1, w.row), chart); w.err != nil {
		return
	}

	w.touch(layoutContentCols)
	w.row += layoutChartRows + 1
}

func (w *layoutWriter) writeText(rb layoutpkg.ResolvedBlock) {
	tb := rb.Block.Text
	if tb == nil {
		w.err = fmt.Errorf("text block %q missing config", rb.Block.ID)

		return
	}

	rendered, err := renderTextTemplate(tb.Text, w.data)
	if err != nil {
		w.err = err

		return
	}

	rendered = strings.TrimSpace(rendered)

	w.mergeRange(1, layoutContentCols, w.row, rendered, w.styles.text)
	w.rowHeight(w.row, textRowHeight(rendered))
	w.row += 2
}

func (w *layoutWriter) writeRawHTML(rb layoutpkg.ResolvedBlock) {
	rh := rb.Block.RawHTML
	if rh == nil {
		w.err = fmt.Errorf("raw_html block %q missing config", rb.Block.ID)

		return
	}

	w.writeNoteText(stripTags(rh.HTML))
}

// sectionText resolves a page header / footer to plain text: raw HTML is
// tag-stripped, a Text template is executed over the dataset.
func (w *layoutWriter) sectionText(sec *models.PageSection) string {
	if sec == nil {
		return ""
	}

	if strings.TrimSpace(sec.HTML) != "" {
		return stripTags(sec.HTML)
	}

	if strings.TrimSpace(sec.Text) == "" {
		return ""
	}

	rendered, err := renderTextTemplate(sec.Text, w.data)
	if err != nil {
		return strings.TrimSpace(sec.Text)
	}

	return strings.TrimSpace(rendered)
}

func (w *layoutWriter) ensureDataSheet() string {
	if w.hasData {
		return layoutDataSheet
	}

	if idx, _ := w.f.GetSheetIndex(layoutDataSheet); idx == -1 {
		if _, err := w.f.NewSheet(layoutDataSheet); err != nil {
			w.err = err

			return layoutDataSheet
		}
	}

	w.hasData = true
	w.dataRow = 1

	return layoutDataSheet
}

func (w *layoutWriter) writeSeriesData(
	sheet string,
	start int,
	rows []map[string]any,
	ch *models.ChartBlock,
) {
	if w.err != nil {
		return
	}

	headers := make([]string, 0, len(ch.Series)+1)
	headers = append(headers, ch.XField)

	for _, s := range ch.Series {
		headers = append(headers, s.TitleOrField())
	}

	for i, h := range headers {
		if w.err = w.f.SetCellValue(sheet, cellName(1+i, start), h); w.err != nil {
			return
		}
	}

	for r, row := range rows {
		if w.err = w.f.SetCellValue(sheet, cellName(1, start+1+r), row[ch.XField]); w.err != nil {
			return
		}

		for s, series := range ch.Series {
			if w.err = w.f.SetCellValue(
				sheet, cellName(2+s, start+1+r), row[series.Field],
			); w.err != nil {
				return
			}
		}
	}

	w.dataRow = start + len(rows) + 3
}

// finish sets column widths from the widest column written, then the page
// layout so a printout fits the page width and never chops columns off.
func (w *layoutWriter) finish(orientation string) {
	if w.err != nil {
		return
	}

	for c := 1; c <= w.maxCol; c++ {
		width := layoutColWidth
		if c == 1 {
			width = layoutFirstColW
		}

		name := cellColumn(c)
		if w.err = w.f.SetColWidth(w.sheet, name, name, width); w.err != nil {
			return
		}
	}

	if w.err = w.f.SetSheetProps(w.sheet, &excelize.SheetPropsOptions{
		FitToPage: boolPtr(true),
	}); w.err != nil {
		return
	}

	size := xlsxPaperA4
	orient := "portrait"

	if strings.EqualFold(strings.TrimSpace(orientation), "landscape") {
		orient = "landscape"
	}

	fitWidth, fitHeight := 1, 0
	w.err = w.f.SetPageLayout(w.sheet, &excelize.PageLayoutOptions{
		Size:        &size,
		Orientation: &orient,
		FitToWidth:  &fitWidth,
		FitToHeight: &fitHeight,
	})
}

func stripTags(s string) string {
	return strings.Join(strings.Fields(htmlTagRE.ReplaceAllString(s, " ")), " ")
}

func textRowHeight(s string) float64 {
	lines := strings.Count(s, "\n") + 1
	if h := float64(lines) * 16; h > 20 {
		return h
	}

	return 20
}

func chartType(kind models.ChartKind, stacked bool) excelize.ChartType {
	switch kind {
	case models.ChartKindBar:
		if stacked {
			return excelize.ColStacked
		}

		return excelize.Col
	case models.ChartKindPie:
		return excelize.Pie
	default:
		return excelize.Line
	}
}

func legendPosition(legend *bool) string {
	if legend != nil && !*legend {
		return "none"
	}

	return "bottom"
}

func chartRef(sheet string, c1, r1, c2, r2 int) string {
	return fmt.Sprintf(
		"'%s'!$%s$%d:$%s$%d",
		sheet, cellColumn(c1), r1, cellColumn(c2), r2,
	)
}

// HTML render tokens, reused here so the workbook matches the html export.
const (
	inkColor       = "2E353B"
	mutedInkColor  = "4C5773"
	headerInkColor = "5B6572"
	subtleInkColor = "949AAB"
	headerFill     = "F4F6F8"
	zebraFill      = "FBFCFD"
	hairline       = "ECEEF1"
	headerRule     = "D9DEE5"
)

var alignByName = map[models.ColumnAlign]int{
	models.ColumnAlignLeft:   0,
	models.ColumnAlignCenter: 1,
	models.ColumnAlignRight:  2,
}

// layoutStyles holds every cell style id the linear renderer needs. The table
// header and body are hand-painted (fill, hairline, zebra) to mirror the html
// table — excelize's built-in table styles carry no dxf, so viewers such as
// LibreOffice render them blank.
type layoutStyles struct {
	title    int
	heading  int
	kpiLabel int
	kpiValue int

	header [3]int    // by alignment: left, center, right
	body   [3][2]int // [alignment][zebra row]

	text int
	note int
}

func newLayoutStyles(f *excelize.File) (layoutStyles, error) {
	mk := func(s *excelize.Style) (int, error) { return f.NewStyle(s) }

	var (
		st  layoutStyles
		err error
	)

	if st.title, err = mk(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 15, Color: inkColor},
	}); err != nil {
		return st, err
	}

	if st.heading, err = mk(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: mutedInkColor},
	}); err != nil {
		return st, err
	}

	if st.kpiLabel, err = mk(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: subtleInkColor},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	}); err != nil {
		return st, err
	}

	if st.kpiValue, err = mk(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 20, Color: inkColor},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	}); err != nil {
		return st, err
	}

	horizontals := [3]string{"left", "center", "right"}

	for i, h := range horizontals {
		if st.header[i], err = mk(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: headerInkColor},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerFill}},
			Border: []excelize.Border{
				{Type: "top", Color: hairline, Style: 1},
				{Type: "bottom", Color: headerRule, Style: 2},
			},
			Alignment: &excelize.Alignment{Horizontal: h, Vertical: "center", WrapText: true},
		}); err != nil {
			return st, err
		}

		for zebra := range 2 {
			style := &excelize.Style{
				Border: []excelize.Border{{Type: "bottom", Color: hairline, Style: 1}},
				Alignment: &excelize.Alignment{
					Horizontal: h, Vertical: "center", WrapText: true,
				},
			}
			if zebra == 1 {
				style.Fill = excelize.Fill{
					Type: "pattern", Pattern: 1, Color: []string{zebraFill},
				}
			}

			if st.body[i][zebra], err = mk(style); err != nil {
				return st, err
			}
		}
	}

	if st.text, err = mk(&excelize.Style{
		Font:      &excelize.Font{Color: inkColor},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "top", WrapText: true},
	}); err != nil {
		return st, err
	}

	if st.note, err = mk(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 9, Color: subtleInkColor},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	}); err != nil {
		return st, err
	}

	return st, nil
}

func renderTextTemplate(templateText string, data models.Dataset) (string, error) {
	funcMap := make(map[string]any, len(textfuncs.FuncMap))
	for k, v := range textfuncs.FuncMap {
		funcMap[k] = v
	}

	t, err := texttemplate.New("layout_text").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)

	return name
}

func cellColumn(col int) string {
	name, _ := excelize.ColumnNumberToName(col)

	return name
}

func cellRange(startCol, startRow, endCol, endRow int) string {
	return fmt.Sprintf("%s:%s", cellName(startCol, startRow), cellName(endCol, endRow))
}

func boolPtr(v bool) *bool { return &v }

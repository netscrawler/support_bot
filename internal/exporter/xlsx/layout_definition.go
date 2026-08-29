package xlsx

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	texttemplate "text/template"
	"support_bot/internal/models"
	"support_bot/internal/pkg/funcs"
	textfuncs "support_bot/internal/pkg/text"

	"github.com/xuri/excelize/v2"
)

type layoutRenderState struct {
	chartDataSheet string
	chartDataRow   int
}

func (e *Exporter) createXlsxBookFromLayout(
	data models.Dataset,
	layout models.Layout,
) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "layout"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, fmt.Errorf("set sheet name: %w", err)
	}

	styles, err := newLayoutStyles(f)
	if err != nil {
		return nil, err
	}

	state := layoutRenderState{}

	for _, block := range layout.Blocks {
		if err := renderLayoutBlock(f, sheetName, data, block, styles, &state); err != nil {
			return nil, fmt.Errorf("render block %q: %w", block.ID, err)
		}
	}

	if state.chartDataSheet != "" {
		if err := f.SetSheetVisible(state.chartDataSheet, false); err != nil {
			return nil, fmt.Errorf("hide chart data sheet: %w", err)
		}
	}

	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(0)

	f.SetAppProps(&excelize.AppProperties{
		Application: "SendyStats",
		Company:     "Sendy",
	})

	return f.WriteToBuffer()
}

type layoutStyles struct {
	header   int
	body     int
	label    int
	value    int
	text     int
	rawHTML  int
}

func newLayoutStyles(f *excelize.File) (layoutStyles, error) {
	header, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "172033"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"D9E2F3"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "D0D5DD", Style: 1},
			{Type: "right", Color: "D0D5DD", Style: 1},
			{Type: "top", Color: "D0D5DD", Style: 1},
			{Type: "bottom", Color: "D0D5DD", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		return layoutStyles{}, err
	}

	body, err := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "E5E9F0", Style: 1},
			{Type: "right", Color: "E5E9F0", Style: 1},
			{Type: "top", Color: "E5E9F0", Style: 1},
			{Type: "bottom", Color: "E5E9F0", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		return layoutStyles{}, err
	}

	label, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "344054"},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	if err != nil {
		return layoutStyles{}, err
	}

	value, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "111827"},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	if err != nil {
		return layoutStyles{}, err
	}

	textStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
	})
	if err != nil {
		return layoutStyles{}, err
	}

	rawHTMLStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "667085"},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
	})
	if err != nil {
		return layoutStyles{}, err
	}

	return layoutStyles{
		header:  header,
		body:    body,
		label:   label,
		value:   value,
		text:    textStyle,
		rawHTML: rawHTMLStyle,
	}, nil
}

func renderLayoutBlock(
	f *excelize.File,
	sheet string,
	data models.Dataset,
	block models.Block,
	styles layoutStyles,
	state *layoutRenderState,
) error {
	switch block.Type {
	case models.BlockTypeTable:
		return renderLayoutTable(f, sheet, data, block, styles)
	case models.BlockTypeChart:
		return renderLayoutChart(f, sheet, data, block, styles, state)
	case models.BlockTypeMetric:
		return renderLayoutMetric(f, sheet, data, block, styles)
	case models.BlockTypeText:
		return renderLayoutText(f, sheet, data, block, styles)
	case models.BlockTypeRawHTML:
		return renderLayoutRawHTML(f, sheet, block, styles)
	default:
		return fmt.Errorf("unsupported block type %q", block.Type)
	}
}

func renderLayoutTable(
	f *excelize.File,
	sheet string,
	data models.Dataset,
	block models.Block,
	styles layoutStyles,
) error {
	if block.Table == nil {
		return fmt.Errorf("table configuration is required")
	}

	rows, ok := data[block.Dataset]
	if !ok {
		return fmt.Errorf("dataset %q not found", block.Dataset)
	}

	if len(rows) == 0 {
		rows = []map[string]any{}
	}

	columns := block.Table.Columns
	fields := make([]string, 0, len(columns))
	for _, column := range columns {
		fields = append(fields, column.Field)
	}

	matrix := pkg.ConvertSortedRows(rows, fields)
	if len(matrix) == 0 {
		matrix = [][]any{make([]any, len(columns))}
		for i, column := range columns {
			matrix[0][i] = column.Title
		}
	}

	startCol, startRow := block.Pos.X+1, block.Pos.Y+1
	if err := writeMatrix(f, sheet, startCol, startRow, matrix, styles.body, styles.header); err != nil {
		return err
	}

	endCol := startCol + len(columns) - 1
	endRow := startRow + len(matrix) - 1
	if endCol < startCol || endRow < startRow {
		return nil
	}

	tableName := sanitizeSheetName(block.ID)
	if tableName == "" {
		tableName = "layout_table"
	}

	if err := f.AddTable(sheet, &excelize.Table{
		Range:             cellRange(startCol, startRow, endCol, endRow),
		Name:              tableName,
		StyleName:         "TableStyleMedium9",
		ShowColumnStripes: false,
		ShowFirstColumn:   false,
		ShowHeaderRow:     boolPtr(true),
		ShowLastColumn:    false,
		ShowRowStripes:    boolPtr(true),
	}); err != nil {
		return fmt.Errorf("add table: %w", err)
	}

	for idx, column := range columns {
		col := startCol + idx
		width := column.Width
		if width <= 0 {
			width = getAutoWidth(matrix, idx)
		}

		colName, _ := excelize.ColumnNumberToName(col)
		if err := f.SetColWidth(sheet, colName, colName, width); err != nil {
			return fmt.Errorf("set column width: %w", err)
		}
	}

	return nil
}

func renderLayoutMetric(
	f *excelize.File,
	sheet string,
	data models.Dataset,
	block models.Block,
	styles layoutStyles,
) error {
	if block.Metric == nil {
		return fmt.Errorf("metric configuration is required")
	}

	rows, ok := data[block.Dataset]
	if !ok {
		return fmt.Errorf("dataset %q not found", block.Dataset)
	}

	value := ""
	for _, row := range rows {
		if v, ok := row[block.Metric.Field]; ok {
			value = fmt.Sprint(v)
			break
		}
	}

	if value == "" && len(rows) == 0 {
		return fmt.Errorf("metric dataset %q is empty", block.Dataset)
	}

	startCol, startRow := block.Pos.X+1, block.Pos.Y+1
	endCol := startCol + max(block.Pos.W, 1) - 1

	label := block.Metric.Label
	if label == "" {
		label = block.Metric.Field
	}

	if err := f.MergeCell(sheet, cellName(startCol, startRow), cellName(endCol, startRow)); err != nil {
		return fmt.Errorf("merge metric label: %w", err)
	}
	if err := f.SetCellValue(sheet, cellName(startCol, startRow), label); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, cellName(startCol, startRow), cellName(endCol, startRow), styles.label); err != nil {
		return err
	}

	if err := f.MergeCell(sheet, cellName(startCol, startRow+1), cellName(endCol, startRow+1)); err != nil {
		return fmt.Errorf("merge metric value: %w", err)
	}
	if err := f.SetCellValue(sheet, cellName(startCol, startRow+1), value); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, cellName(startCol, startRow+1), cellName(endCol, startRow+1), styles.value); err != nil {
		return err
	}

	return nil
}

func renderLayoutText(
	f *excelize.File,
	sheet string,
	data models.Dataset,
	block models.Block,
	styles layoutStyles,
) error {
	if block.Text == nil {
		return fmt.Errorf("text configuration is required")
	}

	rendered, err := renderTextTemplate(block.Text.Text, data)
	if err != nil {
		return err
	}

	startCol, startRow := block.Pos.X+1, block.Pos.Y+1
	endCol := startCol + max(block.Pos.W, 1) - 1
	if err := f.MergeCell(sheet, cellName(startCol, startRow), cellName(endCol, startRow)); err != nil {
		return fmt.Errorf("merge text block: %w", err)
	}

	if err := f.SetCellValue(sheet, cellName(startCol, startRow), rendered); err != nil {
		return err
	}

	return f.SetCellStyle(sheet, cellName(startCol, startRow), cellName(endCol, startRow), styles.text)
}

func renderLayoutRawHTML(
	f *excelize.File,
	sheet string,
	block models.Block,
	styles layoutStyles,
) error {
	if block.RawHTML == nil {
		return fmt.Errorf("raw_html configuration is required")
	}

	startCol, startRow := block.Pos.X+1, block.Pos.Y+1
	endCol := startCol + max(block.Pos.W, 1) - 1

	if err := f.MergeCell(sheet, cellName(startCol, startRow), cellName(endCol, startRow)); err != nil {
		return fmt.Errorf("merge raw_html block: %w", err)
	}

	if err := f.SetCellValue(sheet, cellName(startCol, startRow), block.RawHTML.HTML); err != nil {
		return err
	}

	return f.SetCellStyle(sheet, cellName(startCol, startRow), cellName(endCol, startRow), styles.rawHTML)
}

func renderLayoutChart(
	f *excelize.File,
	mainSheet string,
	data models.Dataset,
	block models.Block,
	styles layoutStyles,
	state *layoutRenderState,
) error {
	if block.Chart == nil {
		return fmt.Errorf("chart configuration is required")
	}

	rows, ok := data[block.Dataset]
	if !ok {
		return fmt.Errorf("dataset %q not found", block.Dataset)
	}

	series := block.Chart.Series
	if len(series) == 0 {
		return fmt.Errorf("chart must have at least one series")
	}

	dataSheet := ensureChartDataSheet(f, state)
	startRow := state.chartDataRow
	startCol := 1

	if err := writeChartData(f, dataSheet, startCol, startRow, rows, block.Chart.XField, series); err != nil {
		return err
	}

	categories := fmt.Sprintf("%s!$%s$%d:$%s$%d",
		dataSheet, cellColumn(startCol), startRow+1, cellColumn(startCol), startRow+len(rows))

	ch := &excelize.Chart{
		Type: chartTypeFromKind(block.Chart.Kind),
		Dimension: excelize.ChartDimension{
			Width:  chartWidth(block.Pos.W),
			Height: chartHeight(block.Pos.H),
		},
		Legend: excelize.ChartLegend{Position: "right"},
	}

	for idx, s := range series {
		valueCol := startCol + idx + 1
		valueRange := fmt.Sprintf("%s!$%s$%d:$%s$%d",
			dataSheet, cellColumn(valueCol), startRow+1, cellColumn(valueCol), startRow+len(rows))
		nameRef := fmt.Sprintf("%s!$%s$%d", dataSheet, cellColumn(valueCol), startRow)
		if idx == 0 && block.Chart.Kind == models.ChartKindPie {
			valueRange = fmt.Sprintf("%s!$%s$%d:$%s$%d",
				dataSheet, cellColumn(valueCol), startRow+1, cellColumn(valueCol), startRow+len(rows))
			categories = fmt.Sprintf("%s!$%s$%d:$%s$%d",
				dataSheet, cellColumn(startCol), startRow+1, cellColumn(startCol), startRow+len(rows))
		}

		ch.Series = append(ch.Series, excelize.ChartSeries{
			Name:       nameRef,
			Categories: categories,
			Values:     valueRange,
		})
		_ = s
	}

	startCell := cellName(block.Pos.X+1, block.Pos.Y+1)
	if err := f.AddChart(mainSheet, startCell, ch); err != nil {
		return fmt.Errorf("add chart: %w", err)
	}

	state.chartDataRow = startRow + len(rows) + 3
	return nil
}

func ensureChartDataSheet(f *excelize.File, state *layoutRenderState) string {
	if state.chartDataSheet != "" {
		return state.chartDataSheet
	}

	const dataSheet = "_layout_data"
	if idx, _ := f.GetSheetIndex(dataSheet); idx == -1 {
		f.NewSheet(dataSheet)
	}

	state.chartDataSheet = dataSheet
	state.chartDataRow = 1
	return dataSheet
}

func writeChartData(
	f *excelize.File,
	sheet string,
	startCol int,
	startRow int,
	rows []map[string]any,
	xField string,
	series []models.Series,
) error {
	headers := make([]string, 0, len(series)+1)
	headers = append(headers, xField)
	for _, s := range series {
		headers = append(headers, s.TitleOrField())
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(startCol+i, startRow)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
	}

	for rowIdx, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(startCol, startRow+rowIdx+1)
		if err := f.SetCellValue(sheet, cell, row[xField]); err != nil {
			return err
		}

		for seriesIdx, s := range series {
			cell, _ := excelize.CoordinatesToCellName(startCol+seriesIdx+1, startRow+rowIdx+1)
			if err := f.SetCellValue(sheet, cell, row[s.Field]); err != nil {
				return err
			}
		}
	}

	return nil
}

func renderTextTemplate(templateText string, data models.Dataset) (string, error) {
	t, err := texttemplate.New("layout_text").
		Funcs(texttemplate.FuncMap(textfuncs.FuncMap)).
		Funcs(funcs.MapJoin(texttemplate.FuncMap{}, texttemplate.FuncMap{})).
		Parse(templateText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func writeMatrix(
	f *excelize.File,
	sheet string,
	startCol, startRow int,
	matrix [][]any,
	bodyStyle, headerStyle int,
) error {
	for rowIdx, row := range matrix {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(startCol+colIdx, startRow+rowIdx)
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return err
			}

			styleID := bodyStyle
			if rowIdx == 0 {
				styleID = headerStyle
			}

			if err := f.SetCellStyle(sheet, cell, cell, styleID); err != nil {
				return err
			}
		}
	}

	return nil
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

func boolPtr(v bool) *bool {
	return &v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func chartTypeFromKind(kind models.ChartKind) excelize.ChartType {
	switch kind {
	case models.ChartKindBar:
		return excelize.Bar
	case models.ChartKindPie:
		return excelize.Pie
	default:
		return excelize.Line
	}
}

func chartWidth(w int) uint {
	if w <= 0 {
		return 480
	}

	return uint(60 * w)
}

func chartHeight(h int) uint {
	if h <= 0 {
		return 300
	}

	return uint(80 * h)
}

func (s models.Series) TitleOrField() string {
	if strings.TrimSpace(s.Title) != "" {
		return s.Title
	}

	return s.Field
}


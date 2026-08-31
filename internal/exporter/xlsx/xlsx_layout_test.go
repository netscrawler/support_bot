package xlsx_test

import (
	"strconv"
	"strings"
	"testing"

	"support_bot/internal/exporter/xlsx"
	"support_bot/internal/models"

	"github.com/xuri/excelize/v2"
)

const layoutSheet = "Отчёт"

func layoutDataset() models.Dataset {
	return models.Dataset{
		"daily": []map[string]any{
			{"date": "2026-08-01", "accepted": 120, "rejected": 10},
			{"date": "2026-08-02", "accepted": 140, "rejected": 12},
			{"date": "2026-08-03", "accepted": 90, "rejected": 8},
		},
	}
}

func openLayout(t *testing.T, out []models.Data) *excelize.File {
	t.Helper()

	if len(out) != 1 {
		t.Fatalf("expected one file, got %d", len(out))
	}

	f, err := excelize.OpenReader(out[0].Data)
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}

	t.Cleanup(func() { _ = f.Close() })

	return f
}

func TestExporterLayout(t *testing.T) {
	name := "layout_report"

	l := models.Layout{
		Version: 1,
		Page: models.PageConfig{
			Header: &models.PageSection{Text: "Отчёт за август"},
		},
		Blocks: []models.Block{
			{
				ID: "m", Type: models.BlockTypeMetric, Dataset: "daily",
				Pos:    models.GridPosition{W: 3, H: 2},
				Metric: &models.MetricBlock{Field: "accepted", Label: "Принято", Format: "number"},
			},
			{
				ID: "c", Type: models.BlockTypeChart, Dataset: "daily",
				Pos: models.GridPosition{W: 8, H: 6},
				Chart: &models.ChartBlock{
					Kind:   models.ChartKindLine,
					XField: "date",
					Title:  "Принято по дням",
					Legend: new(false),
					Series: []models.Series{{Field: "accepted", Title: "Принято"}},
				},
			},
			{
				ID: "t", Type: models.BlockTypeTable, Dataset: "daily",
				Pos: models.GridPosition{W: 12, H: 0},
				Table: &models.TableBlock{
					Columns: []models.Column{
						{Field: "date", Title: "Дата"},
						{Field: "accepted", Title: "Принято"},
						{Field: "rejected", Title: "Отклонено"},
					},
					Sort: []models.TableSort{{Field: "accepted", Desc: true}},
				},
			},
		},
	}

	f := openLayout(t, mustExport(t, layoutDataset(), name, l))

	sheets := f.GetSheetList()
	if len(sheets) != 2 || sheets[0] != layoutSheet {
		t.Fatalf("want [%q _source], got %v", layoutSheet, sheets)
	}

	if visible, _ := f.GetSheetVisible("_source"); visible {
		t.Errorf("_source sheet should be hidden")
	}

	// One continuous table, header from Column.Title (not the field name).
	tables, err := f.GetTables(layoutSheet)
	if err != nil || len(tables) != 1 {
		t.Fatalf("expected one excelize table, got %d (err %v)", len(tables), err)
	}

	hdr := cellFinder(t, f, "Дата")
	if got, _ := f.GetCellValue(layoutSheet, hdr); got != "Дата" {
		t.Fatalf("table header not the column title: %q", got)
	}

	// Sort desc by accepted -> first data row is 2026-08-02 (140), directly
	// under the "Дата" header.
	col, row := splitCell(t, hdr)
	first, _ := f.GetCellValue(layoutSheet, col+strconv.Itoa(row+1))
	if first != "2026-08-02" {
		t.Errorf("table not sorted by accepted desc: first row = %q", first)
	}

	// The metric value is formatted through layout.FormatValue.
	if cells, _ := f.SearchSheet(layoutSheet, "140"); len(cells) == 0 {
		t.Errorf("metric value 140 not found on the sheet")
	}
}

func TestExporterLayoutSingleContinuousTable(t *testing.T) {
	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	l := models.Layout{Blocks: []models.Block{{
		ID: "table", Type: models.BlockTypeTable, Dataset: "rows",
		Pos:   models.GridPosition{W: 12},
		Table: &models.TableBlock{Columns: []models.Column{{Field: "n", Title: "N"}}},
	}}}

	f := openLayout(t, mustExport(t, models.Dataset{"rows": rows}, "pages", l))

	// A linear workbook must not split a table into chunks.
	tables, err := f.GetTables(layoutSheet)
	if err != nil || len(tables) != 1 {
		t.Fatalf("want one continuous table, got %d (err %v)", len(tables), err)
	}

	// 1 header + 200 data rows, contiguous.
	if got, _ := f.GetCellValue(layoutSheet, "A201"); got == "" {
		t.Fatal("expected 200 contiguous data rows")
	}

	if got, _ := f.GetCellValue(layoutSheet, "A202"); got != "" {
		t.Errorf("table runs longer than its data: A202 = %q", got)
	}
}

func TestExporterLayoutFormatsAndAligns(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "ch",
		Pos: models.GridPosition{W: 12, H: 0},
		Table: &models.TableBlock{Columns: []models.Column{
			{Field: "channel", Title: "Канал", Align: models.ColumnAlignLeft},
			{Field: "share", Title: "Доля", Align: models.ColumnAlignRight, Format: "percent"},
		}},
	}}}

	data := models.Dataset{"ch": {{"channel": "Веб", "share": 0.46}}}
	f := openLayout(t, mustExport(t, data, "fmt", l))

	cells, err := f.SearchSheet(layoutSheet, "46%")
	if err != nil || len(cells) == 0 {
		t.Fatalf("formatted percent value not found (err %v, cells %v)", err, cells)
	}

	styleID, err := f.GetCellStyle(layoutSheet, cells[0])
	if err != nil {
		t.Fatalf("get cell style: %v", err)
	}

	st, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("get style: %v", err)
	}

	if st.Alignment == nil || st.Alignment.Horizontal != "right" {
		t.Errorf("share cell not right-aligned: %+v", st.Alignment)
	}
}

func TestExporterLayoutConditionalFormat(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "ch",
		Pos: models.GridPosition{W: 12, H: 0},
		Table: &models.TableBlock{
			Title: "Каналы",
			Columns: []models.Column{
				{Field: "channel", Title: "Канал", Align: models.ColumnAlignLeft},
				{Field: "share", Title: "Доля", Align: models.ColumnAlignRight, Format: "percent"},
			},
			Conditional: []models.CondRule{{
				When:    "value < 0.1",
				Target:  models.CondTargetCell,
				Columns: []string{"share"},
				Style:   models.CondStyleBad,
			}},
		},
	}}}

	data := models.Dataset{"ch": {
		{"channel": "Веб", "share": 0.6},
		{"channel": "Почта", "share": 0.07},
	}}

	f := openLayout(t, mustExport(t, data, "cf", l))

	low := cellFinder(t, f, "7%")
	styleID, err := f.GetCellStyle(layoutSheet, low)
	if err != nil {
		t.Fatalf("get cell style: %v", err)
	}

	st, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("get style: %v", err)
	}

	if st.Fill.Type != "pattern" || len(st.Fill.Color) == 0 ||
		!strings.EqualFold(st.Fill.Color[0], "FDECEA") {
		t.Errorf("bad-rule cell not filled with the preset colour: %+v", st.Fill)
	}

	high := cellFinder(t, f, "60%")
	highStyleID, _ := f.GetCellStyle(layoutSheet, high)
	highSt, _ := f.GetStyle(highStyleID)

	if len(highSt.Fill.Color) > 0 && strings.EqualFold(highSt.Fill.Color[0], "FDECEA") {
		t.Error("the healthy cell was coloured by the rule")
	}
}

func mustExport(t *testing.T, data models.Dataset, name string, l models.Layout) []models.Data {
	t.Helper()

	out, err := xlsx.Exporter{}.Export(data, models.Export{FileName: &name, Layout: &l})
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	return out
}

func cellFinder(t *testing.T, f *excelize.File, needle string) string {
	t.Helper()

	cells, err := f.SearchSheet(layoutSheet, needle)
	if err != nil || len(cells) == 0 {
		t.Fatalf("cell %q not found (err %v)", needle, err)
	}

	return cells[0]
}

func splitCell(t *testing.T, cell string) (string, int) {
	t.Helper()

	col, row, err := excelize.SplitCellName(cell)
	if err != nil {
		t.Fatalf("split cell %q: %v", cell, err)
	}

	return col, row
}

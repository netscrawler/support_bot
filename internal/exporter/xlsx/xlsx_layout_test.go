package xlsx_test

import (
	"testing"

	"support_bot/internal/exporter/xlsx"
	"support_bot/internal/models"

	"github.com/xuri/excelize/v2"
)

func layoutDataset() models.Dataset {
	return models.Dataset{
		"daily": []map[string]any{
			{"date": "2026-08-01", "accepted": 120, "rejected": 10},
			{"date": "2026-08-02", "accepted": 140, "rejected": 12},
			{"date": "2026-08-03", "accepted": 90, "rejected": 8},
		},
	}
}

func TestExporterLayout(t *testing.T) {
	name := "layout_report"

	l := models.Layout{
		Version: 1,
		Blocks: []models.Block{
			{
				ID: "m", Type: models.BlockTypeMetric, Dataset: "daily",
				Pos:    models.GridPosition{X: 0, Y: 0, W: 3, H: 2},
				Metric: &models.MetricBlock{Field: "accepted", Label: "Принято"},
			},
			{
				ID: "c", Type: models.BlockTypeChart, Dataset: "daily",
				Pos: models.GridPosition{X: 4, Y: 0, W: 8, H: 6},
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
				Pos: models.GridPosition{X: 0, Y: 8, W: 12, H: 0},
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

	out, err := xlsx.Exporter{}.Export(layoutDataset(), models.Export{FileName: &name, Layout: &l})
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("expected one file, got %d", len(out))
	}

	f, err := excelize.OpenReader(out[0].Data)
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	hasLayout, hasData := false, false
	for _, s := range sheets {
		switch s {
		case "layout":
			hasLayout = true
		case "_layout_data":
			hasData = true
		}
	}

	if !hasLayout {
		t.Fatalf("missing 'layout' sheet, got %v", sheets)
	}

	if !hasData {
		t.Fatalf("missing hidden '_layout_data' sheet, got %v", sheets)
	}

	if visible, _ := f.GetSheetVisible("_layout_data"); visible {
		t.Errorf("_layout_data sheet should be hidden")
	}

	tables, err := f.GetTables("layout")
	if err != nil || len(tables) != 1 {
		t.Fatalf("expected one excelize table on 'layout', got %d (err %v)", len(tables), err)
	}

	// Sort desc by accepted -> first data row of the table is date 2026-08-02 (140).
	// The packed table starts at grid row 6. Each grid row maps to six worksheet
	// rows, so its header is on row 37 and the first data row is on row 38.
	got, _ := f.GetCellValue("layout", "A38")
	if got != "2026-08-02" {
		t.Errorf("table not sorted by accepted desc: A38 = %q, want 2026-08-02", got)
	}
}

func TestExporterLayoutAppendsPackedPagesOnOneSheet(t *testing.T) {
	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	l := models.Layout{Blocks: []models.Block{{
		ID: "table", Type: models.BlockTypeTable, Dataset: "rows",
		Pos:   models.GridPosition{W: 12},
		Table: &models.TableBlock{Columns: []models.Column{{Field: "n", Title: "N"}}},
	}}}

	out, err := xlsx.Exporter{}.Export(
		models.Dataset{"rows": rows},
		models.Export{FileName: new("pages"), Layout: &l},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	f, err := excelize.OpenReader(out[0].Data)
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer f.Close()

	// The final packed chunk starts after the first 12x6 grid rows and the
	// configured page gap, so data must continue below the first visual page.
	got, err := f.GetCellValue("layout", "A80")
	if err != nil {
		t.Fatalf("read continuation row: %v", err)
	}
	if got == "" {
		t.Fatal("expected packed continuation on the same worksheet")
	}
}

package html_test

import (
	"strings"
	"testing"

	"support_bot/internal/exporter/html"
	"support_bot/internal/models"
)

func TestExportLayoutAllBlockTypes(t *testing.T) {
	data := models.Dataset{
		"daily": {
			{"date": "2026-08-01", "accepted": 120, "rejected": 10},
			{"date": "2026-08-02", "accepted": 140, "rejected": 12},
		},
	}

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
				Pos: models.GridPosition{X: 3, Y: 0, W: 9, H: 4},
				Chart: &models.ChartBlock{
					Kind:   models.ChartKindLine,
					XField: "date",
					Series: []models.Series{
						{Field: "accepted", Title: "Принято"},
						{Field: "rejected", Title: "Отклонено"},
					},
				},
			},
			{
				ID: "t", Type: models.BlockTypeTable, Dataset: "daily",
				Pos: models.GridPosition{X: 0, Y: 4, W: 12, H: 0},
				Table: &models.TableBlock{Columns: []models.Column{
					{Field: "date", Title: "Дата"},
					{Field: "accepted", Title: "Принято"},
				}},
			},
			{
				ID: "x", Type: models.BlockTypeText,
				Pos:  models.GridPosition{X: 0, Y: 8, W: 12, H: 0},
				Text: &models.TextBlock{Text: `{{ printf "rows=%d" (len .daily) }}`},
			},
			{
				ID: "r", Type: models.BlockTypeRawHTML,
				Pos:     models.GridPosition{X: 0, Y: 10, W: 12, H: 2},
				RawHTML: &models.RawHTMLBlock{HTML: `<p id="raw">hi</p>`},
			},
		},
	}

	name := "layout_report"
	out, err := html.Exporter{}.Export(data, models.Export{Layout: &l, FileName: &name})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("want 1 file, got %d", len(out))
	}

	got := out[0].Data.String()
	for _, want := range []string{
		`class="report-page"`,
		`class="report-grid-item"`,
		`"type":"line"`,
		`class="metric-value"`,
		"Принято",
		`class="report-text"`,
		"rows=2",
		`<p id="raw">hi</p>`,
		"2026-08-01",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}

	if !strings.HasSuffix(out[0].FileName, ".html") {
		t.Errorf("unexpected file name %q", out[0].FileName)
	}
}

func TestExportLayoutPaginatesTallTable(t *testing.T) {
	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	l := models.Layout{
		Version: 1,
		Blocks: []models.Block{
			{
				ID: "t", Type: models.BlockTypeTable, Dataset: "big",
				Pos:   models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Table: &models.TableBlock{Columns: []models.Column{{Field: "n", Title: "N"}}},
			},
		},
	}

	name := "pg"
	out, err := html.Exporter{}.Export(models.Dataset{"big": rows}, models.Export{Layout: &l, FileName: &name})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if pages := strings.Count(out[0].Data.String(), `class="report-page"`); pages < 2 {
		t.Fatalf("expected tall table to paginate, got %d pages", pages)
	}
}

func TestExportLayoutRendersHeaderAndFooter(t *testing.T) {
	data := models.Dataset{
		"daily": {{"date": "2026-08-01", "n": 1}, {"date": "2026-08-02", "n": 2}},
	}

	l := models.Layout{
		Version: 1,
		Page: models.PageConfig{
			Header: &models.PageSection{Text: `Отчёт за {{ len .daily }} дн.`},
			Footer: &models.PageSection{HTML: `<span id="pg">стр. 1</span>`},
		},
		Blocks: []models.Block{
			{
				ID: "t", Type: models.BlockTypeTable, Dataset: "daily",
				Pos:   models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Table: &models.TableBlock{Columns: []models.Column{{Field: "date", Title: "Дата"}}},
			},
		},
	}

	out, err := html.Exporter{}.Export(
		data,
		models.Export{Layout: &l, FileName: new("chrome")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	for _, want := range []string{
		`<header class="report-header">`,
		"Отчёт за 2 дн.",
		`<footer class="report-footer">`,
		`<span id="pg">стр. 1</span>`,
		"display: flex; flex-direction: column; padding: 16mm;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestExportLayoutRespectsLimit(t *testing.T) {
	rows := make([]map[string]any, 50)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	limit := 5
	l := models.Layout{
		Version: 1,
		Blocks: []models.Block{
			{
				ID: "t", Type: models.BlockTypeTable, Dataset: "big",
				Pos: models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Table: &models.TableBlock{
					Columns: []models.Column{{Field: "n", Title: "N"}},
					Limit:   &limit,
				},
			},
		},
	}

	name := "lim"
	out, err := html.Exporter{}.Export(models.Dataset{"big": rows}, models.Export{Layout: &l, FileName: &name})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	// rows 0..4 present, row 5 (and up) not rendered
	if !strings.Contains(got, ">4<") || strings.Contains(got, ">5<") {
		t.Fatalf("limit not applied: want rows 0-4 only")
	}
}

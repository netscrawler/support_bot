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
		`class="report-row"`,
		`class="report-grid-item"`,
		`class="card metric-card"`,
		`class="metric-value"`,
		"Принято",
		`"type":"line"`,
		`class="report-table"`,
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

func TestExportLayoutWrapsAndFillsRow(t *testing.T) {
	// Two charts of w=8 and w=6 cannot share a row: the 8 stretches to full
	// width on row 1, the 6 wraps and stretches to full width on row 2.
	mk := func(id string, w int) models.Block {
		return models.Block{
			ID: id, Type: models.BlockTypeChart, Dataset: "d",
			Pos: models.GridPosition{W: w, H: 4},
			Chart: &models.ChartBlock{
				Kind: models.ChartKindBar, XField: "x",
				Series: []models.Series{{Field: "y"}},
			},
		}
	}

	l := models.Layout{Version: 1, Blocks: []models.Block{mk("a", 8), mk("b", 6)}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"d": {{"x": "a", "y": 1}}},
		models.Export{Layout: &l, FileName: new("wrap")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()

	if n := strings.Count(got, `class="report-row"`); n != 2 {
		t.Fatalf("want 2 flow rows, got %d", n)
	}

	if n := strings.Count(got, `flex-grow: 12;`); n != 2 {
		t.Fatalf("both wrapped blocks should fill the width (flex-grow: 12), got %d", n)
	}
}

func TestExportLayoutTableAlignmentAndFormat(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "ch",
		Pos: models.GridPosition{W: 12, H: 0},
		Table: &models.TableBlock{Columns: []models.Column{
			{Field: "channel", Title: "Канал", Align: models.ColumnAlignLeft},
			{Field: "share", Title: "Доля", Align: models.ColumnAlignRight, Format: "percent"},
		}},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"ch": {{"channel": "Веб", "share": 0.46}}},
		models.Export{Layout: &l, FileName: new("tbl")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	for _, want := range []string{
		`<th class="align-right col-num">Доля</th>`,
		`<td class="align-right col-num">46%</td>`,
		`<td class="align-left">Веб</td>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestExportLayoutTableAutoAlignsAndSizesNumericColumns(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "d",
		Pos: models.GridPosition{W: 12, H: 0},
		// "name" is text -> align-left; "count" is numeric with no explicit
		// align -> align-right + col-num.
		Table: &models.TableBlock{Columns: []models.Column{
			{Field: "name", Title: "Имя"},
			{Field: "count", Title: "Всего", Format: "number"},
		}},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"d": {{"name": "web", "count": 1234}}},
		models.Export{Layout: &l, FileName: new("num")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	for _, want := range []string{
		`<th class="align-left">Имя</th>`,
		`<th class="align-right col-num">Всего</th>`,
		`<td class="align-right col-num">1 234</td>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestExportLayoutTableHonorsColumnWidth(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "d",
		Pos: models.GridPosition{W: 12, H: 0},
		Table: &models.TableBlock{Columns: []models.Column{
			{Field: "a", Title: "A", Width: 160},
			{Field: "b", Title: "B"},
		}},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"d": {{"a": "x", "b": "y"}}},
		models.Export{Layout: &l, FileName: new("w")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	for _, want := range []string{`<colgroup>`, `<col style="width: 160px">`, `<col>`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestExportLayoutTableRendersTitle(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "d",
		Pos: models.GridPosition{W: 12, H: 0},
		Table: &models.TableBlock{
			Title:   "Каналы обращений",
			Columns: []models.Column{{Field: "a", Title: "A"}},
		},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"d": {{"a": "x"}}},
		models.Export{Layout: &l, FileName: new("title")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	if !strings.Contains(got, `<div class="card table-card">`) {
		t.Error("titled table missing the table-card class")
	}

	if !strings.Contains(got, `<h3 class="card-title">Каналы обращений</h3>`) {
		t.Error("table title not rendered as a card heading")
	}
}

func TestExportLayoutTableConditionalFormat(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "d",
		Pos: models.GridPosition{W: 12, H: 0},
		Table: &models.TableBlock{
			Columns: []models.Column{
				{Field: "channel", Title: "Канал"},
				{Field: "share", Title: "Доля", Format: "percent"},
			},
			Conditional: []models.CondRule{
				{
					When:   "has(row.share) && row.share < 0.2",
					Target: models.CondTargetRow,
					Style:  models.CondStyleWarning,
				},
				{
					When:    "value < 0.1",
					Target:  models.CondTargetCell,
					Columns: []string{"share"},
					Style:   models.CondStyleBad,
				},
			},
		},
	}}}

	data := models.Dataset{"d": {
		{"channel": "Веб", "share": 0.60},
		{"channel": "Почта", "share": 0.08},
	}}

	out, err := html.Exporter{}.Export(data, models.Export{Layout: &l, FileName: new("cf")})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()

	// The low-share row carries the warning background...
	if !strings.Contains(got, `<tr style="background:#FFF4E5;">`) {
		t.Error("row rule did not colour the low-share row")
	}

	// ...and its share cell is overridden by the more specific bad rule.
	if !strings.Contains(got, `background:#FDECEA;color:#B42318;`) {
		t.Error("cell rule did not out-rank the row rule on the share cell")
	}

	// The healthy row is untouched.
	if strings.Count(got, "<tr>") < 1 {
		t.Error("a row with no matching rule should have a plain <tr>")
	}
}

func TestExportLayoutConditionalRowsSeesFullSetAcrossPages(t *testing.T) {
	rows := make([]map[string]any, 120)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}
	// The maximum (n = 119) lands well past the first page.

	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "t", Type: models.BlockTypeTable, Dataset: "big",
		Pos: models.GridPosition{W: 12, H: 0},
		Table: &models.TableBlock{
			Columns: []models.Column{{Field: "n", Title: "N", Format: "number"}},
			Conditional: []models.CondRule{{
				When:    "rows.all(r, r.n <= value)",
				Target:  models.CondTargetCell,
				Columns: []string{"n"},
				Style:   models.CondStyleGood,
			}},
		},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"big": rows},
		models.Export{Layout: &l, FileName: new("cfpg")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()

	if pages := strings.Count(got, `class="report-page"`); pages < 2 {
		t.Fatalf("expected the table to paginate, got %d page(s)", pages)
	}

	// Exactly one cell (the global maximum) is highlighted, proving `rows`
	// bound to the full set rather than the current page window.
	if n := strings.Count(got, "background:#E6F4EA;"); n != 1 {
		t.Fatalf("want the single global-max cell highlighted, got %d", n)
	}

	if !strings.Contains(got, `background:#E6F4EA;color:#1E7E34;">119</td>`) {
		t.Error("the highlighted cell is not the global maximum (119)")
	}
}

func TestExportLayoutMetricFormat(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "m", Type: models.BlockTypeMetric, Dataset: "kpi",
		Pos:    models.GridPosition{W: 3, H: 2},
		Metric: &models.MetricBlock{Field: "rate", Label: "Доля", Format: "percent"},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"kpi": {{"rate": 0.895}}},
		models.Export{Layout: &l, FileName: new("m")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if !strings.Contains(out[0].Data.String(), `<div class="metric-value">89.5%</div>`) {
		t.Fatal("metric value not formatted as percent")
	}
}

func TestExportLayoutChartSeriesColours(t *testing.T) {
	l := models.Layout{Version: 1, Blocks: []models.Block{{
		ID: "c", Type: models.BlockTypeChart, Dataset: "d",
		Pos: models.GridPosition{W: 12, H: 4},
		Chart: &models.ChartBlock{
			Kind: models.ChartKindLine, XField: "x",
			Series: []models.Series{
				{Field: "a", Color: "#2f80ed"},
				{Field: "b"}, // unset -> palette
			},
		},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"d": {{"x": "1", "a": 1, "b": 2}}},
		models.Export{Layout: &l, FileName: new("c")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	if !strings.Contains(got, `"#2f80ed"`) {
		t.Error("explicit series colour not applied")
	}

	if !strings.Contains(got, `"#88BF4D"`) {
		t.Error("unset series should fall back to the palette")
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

	out, err := html.Exporter{}.Export(
		models.Dataset{"big": rows},
		models.Export{Layout: &l, FileName: &name},
	)
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
		`<div class="report-page" style="padding: 16mm;">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestExportLayoutHeaderTextEscapesData(t *testing.T) {
	data := models.Dataset{
		"daily": {{"src": `<script>alert(1)</script>`, "n": 1}},
	}

	l := models.Layout{
		Version: 1,
		Page: models.PageConfig{
			Header: &models.PageSection{Text: `Источник: {{ (index .daily 0).src }}`},
		},
		Blocks: []models.Block{
			{
				ID: "t", Type: models.BlockTypeTable, Dataset: "daily",
				Pos:   models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Table: &models.TableBlock{Columns: []models.Column{{Field: "n", Title: "N"}}},
			},
		},
	}

	out, err := html.Exporter{}.Export(data, models.Export{Layout: &l, FileName: new("esc")})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Fatal("header rendered a raw <script> tag from dataset content")
	}

	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("expected the dataset value to be html-escaped in the header")
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

	out, err := html.Exporter{}.Export(
		models.Dataset{"big": rows},
		models.Export{Layout: &l, FileName: &name},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()
	// rows 0..4 present, row 5 (and up) not rendered
	if !strings.Contains(got, ">4</td>") || strings.Contains(got, ">5</td>") {
		t.Fatalf("limit not applied: want rows 0-4 only")
	}
}

func TestExportLayoutChartHeightPx(t *testing.T) {
	l := models.Layout{Blocks: []models.Block{{
		ID: "chart", Type: models.BlockTypeChart, Dataset: "daily",
		Pos: models.GridPosition{X: 0, Y: 0, W: 12, H: 2},
		Chart: &models.ChartBlock{
			Kind: models.ChartKindLine, XField: "date", HeightPx: 900,
			Series: []models.Series{{Field: "n"}},
		},
	}}}

	out, err := html.Exporter{}.Export(
		models.Dataset{"daily": {{"date": "2026-08-01", "n": 1}}},
		models.Export{Layout: &l, FileName: new("chart")},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// height_px is the canvas min-height: the floor for the packer's estimate,
	// and the chart grows past it only into a row stretched by a taller sibling.
	if !strings.Contains(out[0].Data.String(), "min-height: 900px;") {
		t.Fatal("chart canvas does not honour height_px")
	}
}

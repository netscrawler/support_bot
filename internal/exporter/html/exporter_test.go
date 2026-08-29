package html_test

import (
	"os"
	"strings"
	"support_bot/internal/exporter/html"
	"support_bot/internal/models"
	"testing"
)

// exampleTemplateData mirrors the data used by example.gohtml.
func exampleTemplateData() models.Dataset {
	return models.Dataset{
		"LineRows": []map[string]any{
			{"date": "2026-08-01", "accepted": 120, "rejected": 10},
			{"date": "2026-08-02", "accepted": 140, "rejected": 12},
		},
		"BarRows": []map[string]any{
			{"country": "UZ", "count": 500},
			{"country": "KG", "count": 300},
		},
		"TableRows": []map[string]any{
			{"status": "Accepted", "count": 100},
			{"status": "Rejected", "count": 20},
			{"status": "Pending", "count": 40},
			{"status": "Cancelled", "count": 5},
			{"status": "Other", "count": 2},
		},
	}
}

func TestExporterExampleTemplate(t *testing.T) {
	tmplData, err := os.ReadFile("./example.gohtml")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	fileName := "example_report"

	out, err := html.Exporter{}.Export(
		exampleTemplateData(),
		models.Export{
			Template: &models.Template{TemplateText: string(tmplData)},
			FileName: &fileName,
		},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if len(out) != 1 || out[0].Data == nil {
		t.Fatalf("unexpected nil output")
	}

	got := out[0].Data.String()

	// The template must produce a laid out grid: two charts collected into
	// one page plus a full-width table, wrapped into report pages.
	for _, expected := range []string{
		`class="report-page"`,
		`class="report-grid"`,
		`class="report-grid-item"`,
		`"type":"line"`,
		`"type":"bar"`,
		"2026-08-01",
		"Принято",
		"Отклонено",
		"Количество",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("expected HTML to contain %q", expected)
		}
	}

	if !strings.HasSuffix(out[0].FileName, ".html") {
		t.Errorf("unexpected file name: %q", out[0].FileName)
	}

	if t.Failed() {
		path := os.TempDir() + "/example_out.html"
		if werr := os.WriteFile(path, []byte(got), 0o600); werr == nil {
			t.Logf("dumped failed output to %s", path)
		}
	}
}

// TestExporterGridPagination checks that a table too tall for one page is
// split across several report pages.
func TestExporterGridPagination(t *testing.T) {
	rows := make([]map[string]any, 0, 200)
	for range 200 {
		rows = append(rows, map[string]any{"status": "Accepted", "count": 100})
	}

	tmpl := `<!doctype html><html><head>{{ styles }}{{ chartJS }}</head><body>
{{ reportGrid }}{{ table .TableRows (splitList "," "status:Статус,count:Количество") (tableOptions "w" 12) }}{{ endReportGrid }}
</body></html>`

	fileName := "pagination_report"

	out, err := html.Exporter{}.Export(
		models.Dataset{
			"TableRows": rows,
		},
		models.Export{Template: &models.Template{TemplateText: tmpl}, FileName: &fileName},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	pages := strings.Count(out[0].Data.String(), `class="report-page"`)
	if pages < 2 {
		t.Fatalf("expected table split into multiple pages, got %d", pages)
	}
}

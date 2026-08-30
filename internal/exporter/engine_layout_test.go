package exporter_test

import (
	"strings"
	"testing"

	"support_bot/internal/exporter"
	"support_bot/internal/models"
)

func TestEngineRendersDeclarativeLayout(t *testing.T) {
	data := models.Dataset{
		"daily": {
			{"date": "2026-08-01", "accepted": 120},
			{"date": "2026-08-02", "accepted": 140},
		},
	}

	l := models.Layout{
		Version: 1,
		Blocks: []models.Block{
			{
				ID: "t", Type: models.BlockTypeTable, Dataset: "daily",
				Pos: models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Table: &models.TableBlock{Columns: []models.Column{
					{Field: "date", Title: "Дата"},
					{Field: "accepted", Title: "Принято"},
				}},
			},
		},
	}

	e := exporter.NewEngine("", nil)

	out, err := e.Export(data, models.Export{
		Format:   "html",
		FileName: new("report"),
		Layout:   &l,
	})
	if err != nil {
		t.Fatalf("html layout export: %v", err)
	}

	got := out[0].Data.String()
	for _, want := range []string{`class="report-page"`, "2026-08-01", "Принято"} {
		if !strings.Contains(got, want) {
			t.Errorf("html output missing %q", want)
		}
	}
}

func TestEngineConvertsLegacyOrderToLayout(t *testing.T) {
	data := models.Dataset{
		"daily": {{"date": "2026-08-01", "count": 5}},
	}

	e := exporter.NewEngine("", nil)

	out, err := e.Export(data, models.Export{
		Format:   "csv",
		FileName: new("report"),
		Order:    map[string][]string{"daily": {"date", "count"}},
	})
	if err != nil {
		t.Fatalf("csv order export: %v", err)
	}

	// Order now flows through the layout pipeline: one file per table block,
	// named by the converted block id rather than the bare dataset key.
	if len(out) != 1 {
		t.Fatalf("want 1 file, got %d", len(out))
	}

	if !strings.HasSuffix(out[0].FileName, ".csv") {
		t.Fatalf("unexpected file name %q", out[0].FileName)
	}

	if got := out[0].Data.String(); !strings.Contains(got, "date,count") {
		t.Fatalf("csv header missing, got %q", got)
	}
}

package text_test

import (
	"strings"
	"testing"

	"support_bot/internal/exporter/text"
	"support_bot/internal/models"
)

func TestExportLayoutRendersEveryTextBlock(t *testing.T) {
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
				ID: "head", Type: models.BlockTypeText,
				Pos:  models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Text: &models.TextBlock{Text: `{{ printf "rows=%d" (len .daily) }}`},
			},
			{
				ID: "foot", Type: models.BlockTypeText,
				Pos:  models.GridPosition{X: 0, Y: 2, W: 12, H: 0},
				Text: &models.TextBlock{Text: `{{ (index .daily 1).date }}/{{ "ok" | upper }}`},
			},
		},
	}

	out, err := text.Exporter{}.Export(data, models.Export{
		Format: models.ReportFormatText,
		Layout: &l,
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if len(out) != 2 {
		t.Fatalf("want 2 text files, got %d", len(out))
	}

	if got := out[0].Data.String(); got != "rows=2" {
		t.Fatalf("block 1 = %q, want %q", got, "rows=2")
	}

	if got := out[1].Data.String(); got != "2026-08-02/OK" {
		t.Fatalf("block 2 = %q, want %q", got, "2026-08-02/OK")
	}
}

func TestExportLegacyTemplateStillRenders(t *testing.T) {
	data := models.Dataset{
		"daily": {{"date": "2026-08-01", "count": 5}},
	}

	out, err := text.Exporter{}.Export(data, models.Export{
		Format: models.ReportFormatText,
		Template: &models.Template{
			Title:        "legacy",
			Type:         "text",
			TemplateText: `total {{ (index .daily 0).count }}`,
		},
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("want 1 file, got %d", len(out))
	}

	if got := strings.TrimSpace(out[0].Data.String()); got != "total 5" {
		t.Fatalf("legacy template output = %q, want %q", got, "total 5")
	}
}

func TestExportRequiresLayoutOrTemplate(t *testing.T) {
	_, err := text.Exporter{}.Export(models.Dataset{}, models.Export{
		Format: models.ReportFormatText,
	})
	if err == nil {
		t.Fatal("expected an error when neither layout nor template is set")
	}
}

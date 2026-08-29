package xlsx_test

import (
	"strings"
	"support_bot/internal/exporter/xlsx"
	"support_bot/internal/models"
	"testing"

	"github.com/xuri/excelize/v2"
)

func sampleDataset() models.Dataset {
	return models.Dataset{
		"daily": []map[string]any{
			{"date": "2026-08-01", "accepted": 120, "rejected": 10},
			{"date": "2026-08-02", "accepted": 140, "rejected": 12},
		},
		"_meta": {
			{"_meta": map[string]any{"date": "2026-08-29"}},
		},
	}
}

// TestExporterLegacyOrder is a golden test of the legacy Order-based export:
// dataset keys become sheets (underscore-prefixed keys skipped), values are
// typed, an excelize table with filters is added.
func TestExporterLegacyOrder(t *testing.T) {
	fileName := "daily_report"
	order := map[string][]string{"daily": {"date", "accepted", "rejected"}}

	out, err := xlsx.Exporter{}.Export(
		sampleDataset(),
		models.Export{FileName: &fileName, Order: order},
	)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("expected single file, got %d", len(out))
	}

	if !strings.HasSuffix(out[0].FileName, ".xlsx") {
		t.Fatalf("unexpected file name %q", out[0].FileName)
	}

	f, err := excelize.OpenReader(out[0].Data)
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) != 1 || sheets[0] != "daily" {
		t.Fatalf("expected single sheet 'daily', got %v", sheets)
	}

	// header row comes from the map keys in Order sequence
	for col, want := range []string{"date", "accepted", "rejected"} {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		got, err := f.GetCellValue("daily", cell)
		if err != nil || got != want {
			t.Fatalf("header %s: got %q want %q (err %v)", cell, got, want, err)
		}
	}

	// typed numeric cell
	v, err := f.GetCellValue("daily", "B2")
	if err != nil || v != "120" {
		t.Fatalf("cell B2: got %q want \"120\" (err %v)", v, err)
	}

	tables, err := f.GetTables("daily")
	if err != nil || len(tables) != 1 {
		t.Fatalf("expected one excelize table on sheet, got %d (err %v)", len(tables), err)
	}
}

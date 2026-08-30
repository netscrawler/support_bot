package csv_test

import (
	"strings"
	"testing"

	"support_bot/internal/exporter/csv"
	"support_bot/internal/models"
)

func TestExportLegacyOneFilePerDatasetKey(t *testing.T) {
	data := models.Dataset{
		"daily": {{"date": "2026-08-01", "count": 5}},
	}

	out, err := csv.Exporter{}.Export(data, models.Export{
		Format:   models.ReportFormatCsv,
		FileName: new("report"),
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("want 1 file, got %d", len(out))
	}

	if out[0].FileName != "report_daily.csv" {
		t.Fatalf("unexpected file name %q", out[0].FileName)
	}

	if got := out[0].Data.String(); !strings.Contains(got, "2026-08-01") {
		t.Fatalf("csv body missing data: %q", got)
	}
}

func TestExportLayoutOneFilePerTableBlockSortedAndLimited(t *testing.T) {
	data := models.Dataset{
		"daily": {
			{"date": "2026-08-01", "accepted": 120},
			{"date": "2026-08-03", "accepted": 90},
			{"date": "2026-08-02", "accepted": 140},
		},
	}

	l := models.Layout{
		Version: 1,
		Blocks: []models.Block{
			{
				ID: "acc", Type: models.BlockTypeTable, Dataset: "daily",
				Pos: models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Table: &models.TableBlock{
					Columns: []models.Column{
						{Field: "date", Title: "Дата"},
						{Field: "accepted", Title: "Принято"},
					},
					Sort:  []models.TableSort{{Field: "accepted", Desc: true}},
					Limit: new(2),
				},
			},
		},
	}

	out, err := csv.Exporter{}.Export(data, models.Export{
		Format:   models.ReportFormatCsv,
		FileName: new("report"),
		Layout:   &l,
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("want 1 file, got %d", len(out))
	}

	if out[0].FileName != "report_acc.csv" {
		t.Fatalf("unexpected file name %q", out[0].FileName)
	}

	got := out[0].Data.String()

	// header uses the declared column order; sort desc on accepted keeps the
	// top two rows (140, 120) and drops 2026-08-03.
	wantLines := []string{"date,accepted", "2026-08-02,140", "2026-08-01,120"}
	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Errorf("csv body missing %q\n---\n%s", want, got)
		}
	}

	if strings.Contains(got, "2026-08-03") {
		t.Errorf("limit not applied, row 2026-08-03 still present:\n%s", got)
	}
}

func TestExportLayoutFormatsValuesAndBlanksNil(t *testing.T) {
	data := models.Dataset{
		"ch": {
			{"channel": "Веб", "requests": 1620, "share": 0.46},
			{"channel": "Тел", "requests": 95}, // share missing
		},
	}

	l := models.Layout{
		Version: 1,
		Blocks: []models.Block{
			{
				ID: "c", Type: models.BlockTypeTable, Dataset: "ch",
				Pos: models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
				Table: &models.TableBlock{Columns: []models.Column{
					{Field: "channel", Title: "Канал"},
					{Field: "requests", Title: "Обращений", Format: "number"},
					{Field: "share", Title: "Доля", Format: "percent"},
				}},
			},
		},
	}

	out, err := csv.Exporter{}.Export(data, models.Export{
		Format:   models.ReportFormatCsv,
		FileName: new("report"),
		Layout:   &l,
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	got := out[0].Data.String()

	for _, want := range []string{
		"channel,requests,share",
		"Веб,1 620,46%",
		"Тел,95,",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("csv body missing %q\n---\n%s", want, got)
		}
	}

	if strings.Contains(got, "<nil>") {
		t.Errorf("csv rendered <nil> for a missing value:\n%s", got)
	}
}

package exporter_test

import (
	"bytes"
	"strings"
	"support_bot/internal/exporter"
	"support_bot/internal/models"
	"testing"
)

func TestEngineUnknownFormat(t *testing.T) {
	e := exporter.NewEngine("", nil)

	_, err := e.Export(models.Dataset{}, models.Export{Format: "docx"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected unknown format error, got %v", err)
	}
}

func TestEngineDispatchesFormats(t *testing.T) {
	fileName := "report"
	dataset := models.Dataset{
		"daily": []map[string]any{{"date": "2026-08-01", "count": 5}},
	}

	e := exporter.NewEngine("", nil)

	// csv uses Order-free path and produces one file per dataset key
	out, err := e.Export(
		dataset,
		models.Export{Format: "csv", FileName: &fileName},
	)
	if err != nil {
		t.Fatalf("csv export: %v", err)
	}

	if len(out) == 0 || !strings.HasSuffix(out[0].FileName, ".csv") {
		t.Fatalf("unexpected csv output %+v", out)
	}
}

func TestEngineCustomExporterOverrides(t *testing.T) {
	e := exporter.NewEngine("", map[string]exporter.Exporter{
		"csv": stubExporter{content: "custom"},
	})

	fileName := "report"
	out, err := e.Export(
		models.Dataset{"daily": {}},
		models.Export{Format: "csv", FileName: &fileName},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if out[0].Data.String() != "custom" {
		t.Fatalf("expected custom exporter output, got %q", out[0].Data.String())
	}
}

type stubExporter struct {
	content string
}

func (s stubExporter) Export(models.Dataset, models.Export) ([]models.Data, error) {
	data, err := models.NewFileData(bytes.NewBufferString(s.content), "stub.csv")

	return []models.Data{data}, err
}

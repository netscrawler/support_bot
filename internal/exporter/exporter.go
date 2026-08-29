package exporter

import (
	"fmt"
	"support_bot/internal/exporter/csv"
	"support_bot/internal/exporter/html"
	"support_bot/internal/exporter/pdf"
	"support_bot/internal/exporter/png"
	"support_bot/internal/exporter/text"
	"support_bot/internal/exporter/xlsx"
	"support_bot/internal/models"
	"support_bot/internal/pkg/funcs"
)

type Exporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

type Engine struct {
	exporters map[string]Exporter
}

// NewEngine builds an engine with default exporters plus optional custom ones.
// path is the browser/pdf binary location used by the pdf exporter (chrome for
// chromium builds, wkhtmltopdf for wkhtmltopdf builds).
func NewEngine(path string, exporters map[string]Exporter) *Engine {
	defaults := map[string]Exporter{
		"csv":  csv.Exporter{},
		"html": html.Exporter{},
		"pdf":  pdf.New(path, html.Exporter{}),
		"png":  png.Exporter{},
		"text": text.Exporter{},
		"xlsx": xlsx.Exporter{},
	}

	return &Engine{
		exporters: funcs.MapJoin(defaults, exporters),
	}
}

func (e *Engine) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	exp, ok := e.exporters[format.Format]
	if !ok {
		return nil, fmt.Errorf("exporter for format: %s not found", format.Format)
	}

	exportResult, err := exp.Export(data, format)
	if err != nil {
		return nil, fmt.Errorf("export %s error: %w", format.Format, err)
	}

	return exportResult, nil
}

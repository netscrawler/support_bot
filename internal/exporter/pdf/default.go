//go:build !chromium && !wkhtmltopdf

package pdf

import (
	"fmt"
	"support_bot/internal/models"
)

type HtmlExporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

type Exporter struct {
	h HtmlExporter
}

func New(string, HtmlExporter) *Exporter {
	return &Exporter{}
}

func (e *Exporter) Export(models.Dataset, models.Export) ([]models.Data, error) {
	return nil, fmt.Errorf("pdf export not available on this build")
}

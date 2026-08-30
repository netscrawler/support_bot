//go:build !chromium && !wkhtmltopdf

package pdf

import (
	"fmt"
	"support_bot/internal/models"
)

type HtmlExporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

// Exporter is the no-op pdf exporter for builds without a pdf backend tag.
type Exporter struct{}

// New accepts the chromium build's constructor arguments and ignores them.
func New(string, HtmlExporter) *Exporter {
	return &Exporter{}
}

func (e *Exporter) Export(models.Dataset, models.Export) ([]models.Data, error) {
	return nil, fmt.Errorf("pdf export not available on this build")
}

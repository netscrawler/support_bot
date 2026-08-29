//go:build wktmltopdf

package pdf

import (
	"fmt"
	"strings"
	"support_bot/internal/models"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

type HtmlExporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

type Exporter struct {
	h HtmlExporter
}

// New creates a wkhtmltopdf-backed pdf exporter. path is the wkhtmltopdf
// binary location; when empty the library resolves it from PATH.
func New(path string, h HtmlExporter) *Exporter {
	if path != "" {
		wkhtmltopdf.SetPath(path)
	}

	return &Exporter{
		h: h,
	}
}

func (e *Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	html, err := e.h.Export(data, format)
	if err != nil {
		return nil, fmt.Errorf("pdf export: %w", err)
	}

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	for _, b := range html {
		page := wkhtmltopdf.NewPageReader(strings.NewReader(b.Data.String()))
		page.DisableJavascript.Set(false)
		page.EnablePlugins.Set(true)
		page.EnableForms.Set(true)
		page.JavascriptDelay.Set(30000)
		page.NoStopSlowScripts.Set(true)
		page.DebugJavascript.Set(true)

		pdfg.AddPage(page)
	}

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(pdfg.Buffer(), *format.FileName+".pdf")
	if err != nil {
		return nil, err
	}

	return []models.Data{fd}, nil
}

//go:build chromium

package pdf

import (
	"bytes"
	"context"
	"fmt"

	"support_bot/internal/exporter/browser"
	"support_bot/internal/models"
)

type HtmlExporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

type Exporter struct {
	h  HtmlExporter
	br *browser.Browser
}

// New creates a chromium-backed pdf exporter. shellPath is the path to the
// chrome/chromium binary; when empty chromedp looks it up itself.
func New(shellPath string, h HtmlExporter) *Exporter {
	return &Exporter{h: h, br: browser.New(shellPath)}
}

func (e *Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	rendered, err := e.h.Export(data, format)
	if err != nil {
		return nil, fmt.Errorf("pdf export: %w", err)
	}

	if len(rendered) == 0 {
		return nil, fmt.Errorf("pdf export: html exporter returned no output")
	}

	pdf, err := e.br.RenderHTMLToPDF(context.Background(), rendered[0].Data.String())
	if err != nil {
		return nil, err
	}

	name := "report"
	if format.FileName != nil {
		name = *format.FileName
	}

	fd, err := models.NewFileData(bytes.NewBuffer(pdf), name+".pdf")
	if err != nil {
		return nil, err
	}

	return []models.Data{fd}, nil
}

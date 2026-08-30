//go:build chromium

package png

import (
	"bytes"
	"context"
	"fmt"

	"support_bot/internal/exporter/browser"
	"support_bot/internal/models"
)

// A report that paginates into more than ~10 PNGs exceeds a Telegram media
// group; all pages are still returned here and the delivery layer buckets them.

type HtmlExporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

// Exporter screenshots the html exporter's output, one PNG per report page.
type Exporter struct {
	h  HtmlExporter
	br *browser.Browser
}

// New creates a chromium-backed png exporter. shellPath is the chrome/chromium
// binary path; when empty chromedp discovers one.
func New(shellPath string, h HtmlExporter) *Exporter {
	return &Exporter{h: h, br: browser.New(shellPath)}
}

func (e *Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	rendered, err := e.h.Export(data, format)
	if err != nil {
		return nil, fmt.Errorf("png export: %w", err)
	}

	if len(rendered) == 0 {
		return nil, fmt.Errorf("png export: html exporter returned no output")
	}

	shots, err := e.br.Screenshot(context.Background(), rendered[0].Data.String())
	if err != nil {
		return nil, err
	}

	if len(shots) == 0 {
		return nil, fmt.Errorf("png export: no pages captured")
	}

	name := "report"
	if format.FileName != nil {
		name = *format.FileName
	}

	out := make([]models.Data, 0, len(shots))
	for i, shot := range shots {
		fileName := name + ".png"
		if len(shots) > 1 {
			fileName = fmt.Sprintf("%s_%d.png", name, i+1)
		}

		dt, err := models.NewImageData(bytes.NewBuffer(shot), fileName)
		if err != nil {
			return nil, err
		}

		out = append(out, dt)
	}

	return out, nil
}

//go:build chromium

package pdf

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"support_bot/internal/models"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type HtmlExporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

type Exporter struct {
	shellPath string
	h         HtmlExporter
}

// New creates a chromium-backed pdf exporter. shellPath is the path to the
// chrome/chromium binary; when empty chromedp looks it up itself.
func New(shellPath string, h HtmlExporter) *Exporter {
	return &Exporter{
		shellPath: shellPath,
		h:         h,
	}
}

func (e *Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	html, err := e.h.Export(data, format)
	if err != nil {
		return nil, fmt.Errorf("pdf export: %w", err)
	}

	allocCtx, allCancel := chromedp.NewExecAllocator(
		context.Background(),
		chromedp.ExecPath(e.shellPath),

		chromedp.Headless,
		// Chrome sandbox requires user namespaces that are typically
		// unavailable in containers and hardened servers.
		chromedp.NoSandbox,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	)
	defer allCancel()

	ctx, cancel := chromedp.NewContext(
		allocCtx,
	)
	defer cancel()

	var pdf []byte

	err = chromedp.Run(
		ctx,

		chromedp.Navigate("data:text/html,"+url.PathEscape(html[0].Data.String())),

		chromedp.WaitReady("body"),

		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error

			pdf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithLandscape(false).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)

			return err
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("generate PDF: %w", err)
	}

	fd, err := models.NewFileData(bytes.NewBuffer(pdf), *format.FileName+".pdf")

	return []models.Data{fd}, err
}

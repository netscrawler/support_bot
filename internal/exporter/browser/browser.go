//go:build chromium

// Package browser drives a headless Chromium instance for the pdf and png
// exporters. It loads a full HTML document into about:blank via
// Page.setDocumentContent — no data: URL, so there is no CDP URL-length limit
// with an inlined Chart.js bundle and large tables — then either prints the
// page to PDF or screenshots each rendered report page.
package browser

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// pageSelector matches one rendered report page in the html exporter output.
const pageSelector = ".report-page"

// Browser renders HTML through headless Chromium. The zero value is unusable;
// construct it with New.
type Browser struct {
	execPath string
}

// New returns a Browser. execPath is the chrome/chromium binary path; an empty
// string lets chromedp discover one on PATH.
func New(execPath string) *Browser {
	return &Browser{execPath: execPath}
}

// RenderHTMLToPDF loads html and prints it to a single PDF, letting the page's
// own @page CSS drive the sheet size and margins.
func (b *Browser) RenderHTMLToPDF(ctx context.Context, html string) ([]byte, error) {
	var pdf []byte

	err := b.run(ctx, html, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error

		pdf, _, err = page.PrintToPDF().
			WithPrintBackground(true).
			WithPreferCSSPageSize(true).
			WithMarginTop(0).
			WithMarginBottom(0).
			WithMarginLeft(0).
			WithMarginRight(0).
			Do(ctx)

		return err
	}))
	if err != nil {
		return nil, fmt.Errorf("browser: render pdf: %w", err)
	}

	return pdf, nil
}

// Screenshot loads html and captures one PNG per ".report-page" element, in
// document order. A document with no such wrapper yields a single full-page
// screenshot.
func (b *Browser) Screenshot(ctx context.Context, html string) ([][]byte, error) {
	var shots [][]byte

	err := b.run(ctx, html, chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		if err := chromedp.Nodes(pageSelector, &nodes, chromedp.ByQueryAll).Do(ctx); err != nil {
			return err
		}

		if len(nodes) == 0 {
			var buf []byte
			if err := chromedp.CaptureScreenshot(&buf).Do(ctx); err != nil {
				return err
			}

			shots = append(shots, buf)

			return nil
		}

		for _, n := range nodes {
			var buf []byte
			if err := chromedp.ScreenshotNodes([]*cdp.Node{n}, 1, &buf).Do(ctx); err != nil {
				return fmt.Errorf("capture page: %w", err)
			}

			shots = append(shots, buf)
		}

		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("browser: screenshot: %w", err)
	}

	return shots, nil
}

// run allocates a fresh headless browser, loads html into about:blank through
// Page.setDocumentContent, waits for the body, then runs actions.
func (b *Browser) run(ctx context.Context, html string, actions ...chromedp.Action) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoSandbox)
	if b.execPath != "" {
		opts = append(opts, chromedp.ExecPath(b.execPath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	runCtx, cancelRun := chromedp.NewContext(allocCtx)
	defer cancelRun()

	load := []chromedp.Action{
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return fmt.Errorf("get frame tree: %w", err)
			}

			return page.SetDocumentContent(tree.Frame.ID, html).Do(ctx)
		}),
		chromedp.WaitReady("body"),
	}

	return chromedp.Run(runCtx, append(load, actions...)...)
}

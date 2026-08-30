//go:build chromium

package exporter_test

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"support_bot/internal/exporter"
	"support_bot/internal/models"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// a4PxAt96DPI is the height of an A4 sheet at the 96 dpi Chromium lays out with;
// a .report-page taller than this makes PrintToPDF split it onto a second sheet.
const a4PxAt96DPI = 1123

// TestFixtureLayoutFitsA4 renders the kitchen-sink fixture to HTML, measures
// every .report-page in headless Chromium, and fails if any is taller than one
// A4 sheet — the signal that the flow packer's millimetre estimate under-fills
// and the pdf page count will balloon. It also checks the pdf page count equals
// the rendered page count when pdfinfo is available.
func TestFixtureLayoutFitsA4(t *testing.T) {
	cp := chromiumPath()
	if cp == "" {
		t.Skip("no chrome/chromium binary for the calibration check")
	}

	dataset, layout := loadExportFixtures(t)

	name := "report"
	engine := exporter.NewEngine(cp, nil)

	htmlOut, err := engine.Export(dataset, models.Export{
		Format:   models.ReportFormatHTML,
		FileName: &name,
		Layout:   &layout,
	})
	if err != nil {
		t.Fatalf("html export: %v", err)
	}

	heights := measureReportPages(t, cp, htmlOut[0].Data.String())
	if len(heights) == 0 {
		t.Fatal("no .report-page elements measured")
	}

	for i, h := range heights {
		if h > a4PxAt96DPI+8 { // a few px of sub-pixel tolerance
			t.Errorf("page %d renders at %dpx, over one A4 sheet (%dpx) — "+
				"lower the millimetre constants in internal/layout/pack.go",
				i, h, a4PxAt96DPI)
		}
	}

	pdfOut, err := engine.Export(dataset, models.Export{
		Format:   models.ReportFormatPdf,
		FileName: &name,
		Layout:   &layout,
	})
	if err != nil {
		if backendUnavailable(err) {
			return
		}

		t.Fatalf("pdf export: %v", err)
	}

	if pages, ok := pdfPageCount(pdfOut[0].Data.Bytes()); ok && pages != len(heights) {
		t.Errorf("pdf has %d pages but the layout renders %d — a page is spilling",
			pages, len(heights))
	}
}

func measureReportPages(t *testing.T, execPath, html string) []int {
	t.Helper()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox, chromedp.ExecPath(execPath))

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	runCtx, cancelRun := chromedp.NewContext(allocCtx)
	defer cancelRun()

	var raw string

	err := chromedp.Run(runCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}

			return page.SetDocumentContent(tree.Frame.ID, html).Do(ctx)
		}),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(
			`JSON.stringify([...document.querySelectorAll('.report-page')]`+
				`.map(p => Math.round(p.getBoundingClientRect().height)))`,
			&raw,
		),
	)
	if err != nil {
		t.Fatalf("measure pages: %v", err)
	}

	var heights []int
	if err := json.Unmarshal([]byte(raw), &heights); err != nil {
		t.Fatalf("decode measurements %q: %v", raw, err)
	}

	return heights
}

// pdfPageCount shells out to pdfinfo; ok is false when it is not installed.
func pdfPageCount(pdf []byte) (int, bool) {
	bin, err := exec.LookPath("pdfinfo")
	if err != nil {
		return 0, false
	}

	cmd := exec.Command(bin, "-")
	cmd.Stdin = strings.NewReader(string(pdf))

	out, err := cmd.Output()
	if err != nil {
		return 0, false
	}

	for _, line := range strings.Split(string(out), "\n") {
		if rest, ok := strings.CutPrefix(line, "Pages:"); ok {
			n, err := strconv.Atoi(strings.TrimSpace(rest))
			if err != nil {
				return 0, false
			}

			return n, true
		}
	}

	return 0, false
}

//go:build chromium

package png_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"support_bot/internal/exporter/html"
	"support_bot/internal/exporter/png"
	"support_bot/internal/models"
)

func chromePath(t *testing.T) string {
	t.Helper()

	for _, c := range []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome",
	} {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}

	t.Skip("no chrome/chromium binary on PATH")

	return ""
}

func TestExportLayoutScreenshotsEachPage(t *testing.T) {
	path := chromePath(t)

	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	l := models.Layout{
		Version: 1,
		Blocks: []models.Block{{
			ID: "t", Type: models.BlockTypeTable, Dataset: "big",
			Pos:   models.GridPosition{X: 0, Y: 0, W: 12, H: 0},
			Table: &models.TableBlock{Columns: []models.Column{{Field: "n", Title: "N"}}},
		}},
	}

	name := "shot"

	out, err := png.New(path, html.Exporter{}).Export(
		models.Dataset{"big": rows},
		models.Export{Format: "png", FileName: &name, Layout: &l},
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	if len(out) < 2 {
		t.Fatalf("expected a 200-row table to paginate into >=2 PNGs, got %d", len(out))
	}

	for i, d := range out {
		b := d.Data.Bytes()
		if !bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")) {
			t.Errorf("page %d is not a PNG (prefix %x)", i, b[:min(8, len(b))])
		}

		if !strings.HasSuffix(d.FileName, ".png") {
			t.Errorf("page %d unexpected name %q", i, d.FileName)
		}
	}
}

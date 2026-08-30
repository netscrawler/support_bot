package exporter_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"support_bot/internal/exporter"
	"support_bot/internal/models"
)

// fixturesDir holds the layout / dataset JSON fixtures; generated files land in
// the git-ignored output/ subtree for manual inspection.
const fixturesDir = "testdata"

// TestExportFixturesToDisk renders the kitchen-sink layout fixture (one block of
// every type, all three chart kinds, page header + footer) into every registered
// format and writes the result under testdata/output/<format>/. It is a manual
// review aid, not a golden test: it fails only when a format errors unexpectedly
// or produces no files. pdf/png are skipped when their browser backend is
// unavailable.
func TestExportFixturesToDisk(t *testing.T) {
	dataset, layout := loadExportFixtures(t)

	fileName := "report"
	engine := exporter.NewEngine(chromiumPath(), nil)

	formats := []string{
		models.ReportFormatCsv,
		models.ReportFormatHTML,
		models.ReportFormatText,
		models.ReportFormatXlsx,
		models.ReportFormatPng,
		models.ReportFormatPdf,
	}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			layout := layout

			out, err := engine.Export(dataset, models.Export{
				Format:   format,
				FileName: &fileName,
				Layout:   &layout,
			})
			if err != nil {
				if backendUnavailable(err) {
					t.Skipf("%s backend unavailable: %v", format, err)
				}

				t.Fatalf("%s export: %v", format, err)
			}

			if len(out) == 0 {
				t.Fatalf("%s export produced no files", format)
			}

			dir := filepath.Join(fixturesDir, "output", format)
			if err := os.RemoveAll(dir); err != nil {
				t.Fatalf("clear %s: %v", dir, err)
			}

			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatalf("mkdir %s: %v", dir, err)
			}

			for i, d := range out {
				name := d.FileName
				if name == "" {
					name = format + "_" + strconv.Itoa(i) + ".txt"
				}

				content := d.Data.Bytes()

				path := filepath.Join(dir, name)
				if err := os.WriteFile(path, content, 0o600); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}

				abs, _ := filepath.Abs(path)
				t.Logf("wrote %s (%d bytes)", abs, len(content))
			}
		})
	}
}

func loadExportFixtures(t *testing.T) (models.Dataset, models.Layout) {
	t.Helper()

	var (
		dataset models.Dataset
		layout  models.Layout
	)

	readJSONFixture(t, filepath.Join(fixturesDir, "dataset.json"), &dataset)
	readJSONFixture(t, filepath.Join(fixturesDir, "layout.json"), &layout)

	return dataset, layout
}

func readJSONFixture(t *testing.T, path string, dst any) {
	t.Helper()

	raw, err := os.ReadFile(path) //nolint:gosec // fixed in-repo fixture path
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}

	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}
}

// backendUnavailable reports whether err is a "no pdf/png backend on this build
// or host" failure rather than a real export regression.
func backendUnavailable(err error) bool {
	msg := strings.ToLower(err.Error())

	for _, marker := range []string{
		"not available",
		"not supported",
		"executable file not found",
		"chrome",
		"chromium",
		"no pages captured",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}

	return false
}

// chromiumPath resolves a chrome/chromium binary for the pdf/png backends,
// preferring an explicit env override. Empty means "let chromedp look it up".
func chromiumPath() string {
	for _, env := range []string{"CHROMIUM_PATH", "CHROME_PATH", "BROWSER_PATH"} {
		if p := os.Getenv(env); p != "" {
			return p
		}
	}

	for _, bin := range []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"chrome",
	} {
		if p, err := exec.LookPath(bin); err == nil {
			return p
		}
	}

	for _, p := range []string{
		"/usr/lib64/chromium-browser/headless_shell",
		"/usr/lib/chromium-browser/headless_shell",
		"/usr/lib/chromium/chromium",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/opt/google/chrome/chrome",
		"/snap/bin/chromium",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

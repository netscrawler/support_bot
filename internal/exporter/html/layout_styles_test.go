package html

import (
	"strings"
	"testing"

	"support_bot/internal/models"
)

func TestLayoutStylesPortrait(t *testing.T) {
	got := string(layoutStyles(models.PageConfig{}))

	for _, want := range []string{
		"@page { size: A4 portrait; margin: 0; }",
		"width: 210mm;",
		"min-height: 293mm;",
		"padding: 16mm;",
		".card {",
		".card-title {",
		".metric-value {",
		".report-row {",
		".report-table th {",
		".align-right { text-align: right; }",
		"break-after: page;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("layoutStyles missing %q", want)
		}
	}

	for _, notWant := range []string{
		// The print block must not clamp a page's height: an overflowing page
		// spills onto another sheet, it is never clipped.
		"max-height:",
		"overflow: hidden",
	} {
		if strings.Contains(got, notWant) {
			t.Errorf("layoutStyles must not contain %q", notWant)
		}
	}

	// The .report-table th rule must not hard-code an alignment, or it would
	// out-specify the per-column .align-* classes on the header cells.
	_, after, ok := strings.Cut(got, ".report-table th {")
	if !ok {
		t.Fatal("layoutStyles has no `.report-table th` rule")
	}

	th, _, ok := strings.Cut(after, "}")
	if !ok {
		t.Fatal("`.report-table th` rule is not closed")
	}

	if strings.Contains(th, "text-align") {
		t.Errorf("`.report-table th` pins text-align, breaking column alignment: %q", th)
	}
}

func TestLayoutStylesLandscapeAndPadding(t *testing.T) {
	got := string(layoutStyles(models.PageConfig{Orientation: "landscape", PaddingMM: 20}))

	for _, want := range []string{
		"@page { size: A4 landscape; margin: 0; }",
		"width: 297mm;",
		"min-height: 206mm;",
		"padding: 20mm;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("layoutStyles missing %q", want)
		}
	}
}

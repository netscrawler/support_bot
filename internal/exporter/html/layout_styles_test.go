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
		"max-height: 297mm;",
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

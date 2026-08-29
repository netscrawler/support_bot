package html

import (
	"strings"
	"testing"

	"support_bot/internal/models"
)

func TestStylesUsesDefaultPageConfig(t *testing.T) {
	got := string(styles())

	for _, expected := range []string{
		"@page {",
		"size: A4 portrait;",
		"width: 210mm;",
		"height: 297mm;",
		"page-break-after: always;",
		"break-after: page;",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected CSS to contain %q", expected)
		}
	}
}

func TestStylesUsesProvidedPageConfig(t *testing.T) {
	got := string(styles(models.PageConfig{
		Format:      "A4",
		Orientation: "landscape",
		PaddingMM:   20,
	}))

	for _, expected := range []string{
		"size: A4 landscape;",
		"width: 297mm;",
		"height: 210mm;",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected CSS to contain %q", expected)
		}
	}
}

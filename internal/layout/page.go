package layout

import (
	"strings"

	"support_bot/internal/models"
)

// A4 is the only sheet size v1 supports. Letter / A3 would add a lookup here.
const (
	a4WidthMM  = 210
	a4HeightMM = 297

	// HeaderBandMM / FooterBandMM is the vertical space reserved above / below
	// the grid when the page declares the matching PageConfig section. The
	// rendered band uses natural (flex) height; these are the work-area
	// estimates used for millimetre sizing by non-CSS renderers.
	HeaderBandMM = 18
	FooterBandMM = 14
)

// PageGeometryFor returns the grid work-area for a page: the A4 sheet minus
// PaddingMM on every side, minus a header / footer band when the matching
// PageConfig section is set. The grid stays a fixed GridColumns x GridRows cell
// matrix — only the millimetre fields vary — so Pack behaves identically
// regardless of page chrome.
func PageGeometryFor(page models.PageConfig) PageGeometry {
	w, h := a4WidthMM, a4HeightMM
	if strings.EqualFold(strings.TrimSpace(page.Orientation), "landscape") {
		w, h = h, w
	}

	pad := page.PaddingMM
	if pad <= 0 {
		pad = DefaultPaddingMM
	}

	workW := w - 2*pad
	workH := h - 2*pad

	if page.Header != nil {
		workH -= HeaderBandMM
	}

	if page.Footer != nil {
		workH -= FooterBandMM
	}

	return PageGeometry{
		Cols:        GridColumns,
		Rows:        GridRows,
		WidthMM:     workW,
		HeightMM:    workH,
		RowHeightMM: float64(workH) / float64(GridRows),
	}
}

package html

import (
	"fmt"
	"html/template"
	"strings"

	"support_bot/internal/models"
)

// fontStack is the shared UI font for the declarative layout render path.
const fontStack = `Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif`

// layoutStyles is the stylesheet for the pure models.Layout render path. It is
// deliberately separate from styles(): that one belongs to the hand-written
// template escape-hatch and must stay byte-stable. The look here is a flat,
// card-based dashboard (white cards on a light ground, muted ink), and nothing
// clips — pages grow past their nominal A4 height rather than hiding overflow.
func layoutStyles(page models.PageConfig) template.HTML {
	orientation := "portrait"
	widthMM, heightMM := 210, 297

	if strings.EqualFold(strings.TrimSpace(page.Orientation), "landscape") {
		orientation = "landscape"
		widthMM, heightMM = heightMM, widthMM
	}

	pad := page.PaddingMM
	if pad <= 0 {
		pad = 16
	}

	// The page box is a few mm under the physical sheet: at 96 dpi a full
	// 297mm min-height rounds up past the @page box and PrintToPDF then splits
	// every page in two. The content never approaches this floor.
	pageMinMM := heightMM - 4

	css := fmt.Sprintf(`<style>
    @page { size: A4 %[1]s; margin: 0; }

    * { box-sizing: border-box; }

    html, body {
        margin: 0;
        background: #f6f7f9;
        color: #2e353b;
        font: 14px/1.5 %[4]s;
    }

    @media screen { body { padding: 24px; } }

    .report-page {
        width: %[2]dmm;
        min-height: %[6]dmm;
        box-sizing: border-box;
        margin: 0 auto;
        padding: %[5]dmm;
        background: #ffffff;
        overflow: visible;
        display: flex;
        flex-direction: column;
    }

    .report-page + .report-page { margin-top: 24px; }

    .report-header {
        flex: 0 0 auto;
        font: 600 13px/1.4 %[4]s;
        color: #4c5773;
        border-bottom: 1px solid #e7eaee;
        padding-bottom: 10px;
        margin-bottom: 14px;
    }

    .report-footer {
        flex: 0 0 auto;
        font: 400 10px/1.4 %[4]s;
        color: #9aa1ab;
        border-top: 1px solid #e7eaee;
        padding-top: 8px;
        margin-top: 14px;
    }

    .report-grid {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
        gap: 14px;
    }

    .report-row {
        display: flex;
        gap: 14px;
        align-items: stretch;
        break-inside: avoid;
        page-break-inside: avoid;
    }

    .report-grid-item {
        display: flex;
        min-width: 0;
        flex-basis: 0;
    }

    .report-grid-item > * { flex: 1 1 auto; min-width: 0; }

    .card {
        background: #ffffff;
        border: 1px solid #e7eaee;
        border-radius: 8px;
        box-shadow: 0 1px 3px rgba(16, 24, 40, 0.06);
        padding: 16px;
        display: flex;
        flex-direction: column;
    }

    .card-title {
        margin: 0 0 12px;
        font: 600 13px/1.3 %[4]s;
        color: #4c5773;
    }

    .metric-card { justify-content: center; }

    .metric-label {
        font: 600 11px/1.2 %[4]s;
        letter-spacing: 0.06em;
        text-transform: uppercase;
        color: #949aab;
        margin-bottom: 6px;
    }

    .metric-value {
        font: 700 30px/1.1 %[4]s;
        color: #2e353b;
    }

    .report-chart {
        position: relative;
        width: 100%%;
    }

    .report-chart canvas {
        width: 100%% !important;
        height: 100%% !important;
    }

    .report-table {
        width: 100%%;
        border-collapse: collapse;
        font-size: 12px;
    }

    .report-table th {
        background: #f9fafb;
        color: #4c5773;
        font-weight: 600;
        padding: 8px 12px;
        border-bottom: 1px solid #e7eaee;
        white-space: nowrap;
        text-align: left;
    }

    .report-table td {
        padding: 7px 12px;
        border-top: 1px solid #f1f3f5;
        color: #2e353b;
    }

    .report-table tbody tr:first-child td { border-top: none; }

    .align-left { text-align: left; }
    .align-center { text-align: center; }
    .align-right { text-align: right; }

    .report-text { font-size: 12px; color: #2e353b; }
    .report-text p { margin: 0 0 8px; }
    .report-text :last-child { margin-bottom: 0; }

    @media print {
        html, body { background: #ffffff; }
        body { padding: 0; }
        .report-page + .report-page { margin-top: 0; }
        .report-page {
            margin: 0;
            box-shadow: none;
            /* Never let a page exceed one physical sheet: with a good height
               estimate this is slack, and if one is off the page clips a few
               mm rather than fragmenting into a second sheet. */
            max-height: %[3]dmm;
            overflow: hidden;
            break-after: page;
            page-break-after: always;
        }
        .report-page:last-child {
            break-after: auto;
            page-break-after: auto;
        }
    }
</style>`, orientation, widthMM, heightMM, fontStack, pad, pageMinMM)

	return template.HTML(css) //nolint:gosec // static stylesheet, no user data
}

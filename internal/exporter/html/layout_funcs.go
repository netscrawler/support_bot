package html

import (
	"fmt"
	"html/template"
	"strings"
)

// gridState holds per-render state for template functions that collect report
// grid blocks. Every Export call creates its own instance: a package-level
// stack would be shared between concurrent generator workers.
type gridState struct {
	stack [][]ReportBlock
}

func newGridState() *gridState {
	return &gridState{}
}

// funcMap exposes grid functions bound to this state for a template.
func (s *gridState) funcMap() map[string]any {
	return map[string]any{
		"reportGrid":    s.reportGrid,
		"endReportGrid": s.endReportGrid,
		"chart":         s.chart,
		"table":         s.table,
		"tableOptions":  tableOptions,
	}
}

// reportGrid initializes collection of blocks for a grid.
func (s *gridState) reportGrid() (string, error) {
	s.stack = append(s.stack, []ReportBlock{})

	return "", nil
}

// endReportGrid finalizes collection, runs layout engine and renders pages HTML.
func (s *gridState) endReportGrid() (template.HTML, error) {
	if len(s.stack) == 0 {
		return "", fmt.Errorf("endReportGrid: no active reportGrid")
	}

	// pop
	idx := len(s.stack) - 1
	blocks := s.stack[idx]
	s.stack = s.stack[:idx]

	engine := NewSimpleLayoutEngine(GridColumns, GridRows)
	pages, err := engine.Layout(reportBlocksClone(blocks))
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	for pi, p := range pages {
		sb.WriteString(`<div class="report-page">`)
		sb.WriteString("\n")

		fmt.Fprintf(&sb, `<div class="report-grid" style="grid-template-rows: repeat(%d, minmax(0, 1fr));">`,
			engine.Rows)
		sb.WriteString("\n")

		for _, pb := range p.Blocks {
			colStart := pb.Position.X + 1
			rowStart := pb.Position.Y + 1
			w := pb.Size.W
			h := pb.Size.H
			style := fmt.Sprintf(
				"grid-column: %d / span %d; grid-row: %d / span %d;",
				colStart,
				w,
				rowStart,
				h,
			)
			// render block content
			fmt.Fprintf(&sb, "<div class=\"report-grid-item\" style=\"%s\">%s</div>",
				style,
				pb.Content)
			sb.WriteString("\n")
		}

		sb.WriteString(`</div>`) // close report-grid
		sb.WriteString("\n")
		sb.WriteString(`</div>`) // close report-page
		if pi != len(pages)-1 {
			sb.WriteString("\n")
		}
	}

	return template.HTML(sb.String()), nil
}

func reportBlocksClone(in []ReportBlock) []ReportBlock {
	out := make([]ReportBlock, len(in))
	copy(out, in)

	return out
}

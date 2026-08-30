package html

import (
	"bytes"
	"fmt"
	"html/template"

	"support_bot/internal/exporter/render"
	layoutpkg "support_bot/internal/layout"
	"support_bot/internal/models"
)

// gridState holds per-render state for the {{ reportGrid }} / {{ chart }} /
// {{ table }} template helpers. Every Export call creates its own instance: a
// package-level stack would be shared between concurrent generator workers.
type gridState struct {
	stack  [][]layoutpkg.ResolvedBlock
	nextID func() string
}

func newGridState() *gridState {
	return &gridState{nextID: render.Counter("report-chart")}
}

// funcMap exposes the grid helpers bound to this state for a template.
func (s *gridState) funcMap() map[string]any {
	return map[string]any{
		"reportGrid":    s.reportGrid,
		"endReportGrid": s.endReportGrid,
		"chart":         s.chart,
		"table":         s.table,
		"tableOptions":  tableOptions,
	}
}

func (s *gridState) renderCtx() *render.RenderContext {
	return &render.RenderContext{
		Funcs:  render.FuncMap(),
		NextID: s.nextID,
	}
}

// reportGrid opens a new block collector.
func (s *gridState) reportGrid() (string, error) {
	s.stack = append(s.stack, []layoutpkg.ResolvedBlock{})

	return "", nil
}

// endReportGrid closes the current collector, packs the collected blocks onto
// 12x12 pages and renders the page HTML.
func (s *gridState) endReportGrid() (template.HTML, error) {
	if len(s.stack) == 0 {
		return "", fmt.Errorf("endReportGrid: no active reportGrid")
	}

	idx := len(s.stack) - 1
	blocks := s.stack[idx]
	s.stack = s.stack[:idx]

	doc := layoutpkg.ResolvedDocument{Blocks: blocks}

	pages, err := layoutpkg.Pack(doc, layoutpkg.PageGeometry{
		Cols: layoutpkg.GridColumns,
		Rows: layoutpkg.GridRows,
	})
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := renderPages(&buf, s.renderCtx(), pages, models.PageConfig{}); err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}

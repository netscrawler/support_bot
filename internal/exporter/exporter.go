package exporter

import (
	"context"
	"fmt"

	"support_bot/internal/exporter/csv"
	"support_bot/internal/exporter/html"
	"support_bot/internal/exporter/pdf"
	"support_bot/internal/exporter/png"
	"support_bot/internal/exporter/render"
	"support_bot/internal/exporter/text"
	"support_bot/internal/exporter/xlsx"
	layoutpkg "support_bot/internal/layout"
	"support_bot/internal/models"
)

// Exporter is the legacy single-call export interface. Custom exporters passed
// to NewEngine implement it, and every built-in renderer still satisfies it as
// well (used for the "nothing declarative to render" fallback).
type Exporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

// Engine dispatches an export request to the renderer registered for its
// format.
type Engine struct {
	renderers map[string]render.Renderer
}

// NewEngine builds an engine with the six built-in renderers plus optional
// custom ones. A custom entry that already implements render.Renderer is used
// as-is; one that only implements the legacy Exporter interface is wrapped so
// it still plugs into the pipeline. path is the browser/pdf binary location
// used by the pdf exporter.
func NewEngine(path string, custom map[string]Exporter) *Engine {
	renderers := map[string]render.Renderer{
		"csv":  csv.Exporter{},
		"html": html.Exporter{},
		"png":  png.Exporter{},
		"text": text.Exporter{},
		"xlsx": xlsx.Exporter{},
		"pdf":  legacyRenderer{format: "pdf", inner: pdf.New(path, html.Exporter{})},
	}

	for format, exp := range custom {
		if r, ok := exp.(render.Renderer); ok {
			renderers[format] = r

			continue
		}

		renderers[format] = legacyRenderer{format: format, inner: exp}
	}

	return &Engine{renderers: renderers}
}

// Export converts the request to a layout via LayoutFromExport, resolves and
// packs it for the target renderer's geometry, then renders. A request with
// nothing declarative to convert (no Layout, Template or Order) is handed to
// the renderer's own legacy Export.
func (e *Engine) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	r, ok := e.renderers[format.Format]
	if !ok {
		return nil, fmt.Errorf("exporter for format: %s not found", format.Format)
	}

	l := layoutpkg.LayoutFromExport(format)

	if len(l.Blocks) == 0 {
		legacy, ok := r.(Exporter)
		if !ok {
			return nil, fmt.Errorf("export %s: nothing to render", format.Format)
		}

		out, err := legacy.Export(data, format)
		if err != nil {
			return nil, fmt.Errorf("export %s error: %w", format.Format, err)
		}

		return out, nil
	}

	if err := layoutpkg.Validate(l, layoutpkg.SortedDatasetKeys(data)); err != nil {
		return nil, fmt.Errorf("export %s: invalid layout: %w", format.Format, err)
	}

	doc, err := layoutpkg.Resolve(l, data)
	if err != nil {
		return nil, fmt.Errorf("export %s: resolve layout: %w", format.Format, err)
	}

	pages, err := layoutpkg.Pack(doc, r.Geometry(doc.Page))
	if err != nil {
		return nil, fmt.Errorf("export %s: pack layout: %w", format.Format, err)
	}

	out, err := r.Render(context.Background(), data, doc, pages, format)
	if err != nil {
		return nil, fmt.Errorf("export %s error: %w", format.Format, err)
	}

	return out, nil
}

// legacyRenderer adapts a legacy Exporter to render.Renderer. It ignores the
// resolved document and delegates to the wrapped Export, which is handed the
// original models.Export and chooses its own path. Used for custom map entries
// and for the pdf exporter until it moves onto the browser package (Step 7).
type legacyRenderer struct {
	format string
	inner  Exporter
}

func (a legacyRenderer) Format() string { return a.format }

func (a legacyRenderer) Geometry(models.PageConfig) layoutpkg.PageGeometry {
	return layoutpkg.PageGeometry{Rows: 0}
}

func (a legacyRenderer) Render(
	_ context.Context,
	data models.Dataset,
	_ layoutpkg.ResolvedDocument,
	_ []layoutpkg.Page,
	format models.Export,
) ([]models.Data, error) {
	return a.inner.Export(data, format)
}

func (a legacyRenderer) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	return a.inner.Export(data, format)
}

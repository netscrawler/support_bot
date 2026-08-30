// Package render holds the format-neutral contract every export renderer
// implements, plus the shared template-function set.
package render

import (
	"context"
	"html/template"
	"maps"
	"strconv"

	"github.com/Masterminds/sprig/v3"

	"support_bot/internal/layout"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"
)

// Renderer turns a resolved, packed layout into deliverable files for one
// output format.
type Renderer interface {
	// Format is the models.Export.Format value this renderer handles.
	Format() string
	// Geometry describes the target surface Pack should lay blocks onto.
	Geometry(page models.PageConfig) layout.PageGeometry
	// Render produces the output files. data is the source dataset (needed by
	// text/metric go-templates); doc is the resolved document; pages is the
	// result of Pack(doc, Geometry(doc.Page)); exp carries format-specific
	// options (FileName, Template, ...).
	Render(
		ctx context.Context,
		data models.Dataset,
		doc layout.ResolvedDocument,
		pages []layout.Page,
		exp models.Export,
	) ([]models.Data, error)
}

// RenderContext carries per-render state shared by individual block renderers.
type RenderContext struct {
	Doc   layout.ResolvedDocument
	Data  models.Dataset
	Funcs template.FuncMap
	// NextID yields deterministic per-render element ids.
	NextID func() string
}

// FuncMap is the shared template-function set for text / metric go-templates:
// all of Sprig plus the project helpers in internal/pkg/text.
func FuncMap() template.FuncMap {
	fm := make(template.FuncMap)
	maps.Copy(fm, sprig.FuncMap())
	maps.Copy(fm, text.FuncMap)

	return fm
}

// Counter returns a deterministic id generator yielding "<prefix>-1",
// "<prefix>-2", ... Use one per Render call so output is reproducible.
func Counter(prefix string) func() string {
	n := 0

	return func() string {
		n++

		return prefix + "-" + strconv.Itoa(n)
	}
}

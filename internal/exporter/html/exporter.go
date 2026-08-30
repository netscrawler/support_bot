package html

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"
	texttemplate "text/template"

	"github.com/Masterminds/sprig/v3"

	"support_bot/internal/exporter/render"
	layoutpkg "support_bot/internal/layout"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"
)

type Exporter struct{}

// Format implements render.Renderer.
func (e Exporter) Format() string { return models.ReportFormatHTML }

// Geometry implements render.Renderer: a fixed 12x12 grid whose millimetre
// work-area is the A4 sheet net of padding and any header / footer band.
func (e Exporter) Geometry(page models.PageConfig) layoutpkg.PageGeometry {
	return layoutpkg.PageGeometryFor(page)
}

// Export renders one export. A declarative layout goes through the shared
// Resolve/Pack/Render pipeline; otherwise the report is a hand-written
// html/template that uses the reportGrid/chart/table helpers (escape-hatch).
func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	if format.Layout != nil {
		return e.exportLayout(data, format)
	}

	return e.exportTemplate(data, format)
}

// Render implements render.Renderer for the pure-layout path.
func (e Exporter) Render(
	_ context.Context,
	data models.Dataset,
	doc layoutpkg.ResolvedDocument,
	pages []layoutpkg.Page,
	format models.Export,
) ([]models.Data, error) {
	var buf bytes.Buffer

	if data == nil {
		data = docData(doc)
	}

	rc := &render.RenderContext{
		Doc:    doc,
		Data:   data,
		Funcs:  render.FuncMap(),
		NextID: render.Counter("report-chart"),
	}

	buf.WriteString("<!doctype html>\n<html lang=\"ru\">\n<head>\n<meta charset=\"utf-8\">\n")
	buf.WriteString(string(styles(doc.Page)))
	buf.WriteString("\n")

	if cjs, err := chartJS(); err == nil {
		buf.WriteString(string(cjs))
		buf.WriteString("\n")
	}

	buf.WriteString("</head>\n<body>\n")

	if err := renderPages(&buf, rc, pages, doc.Page); err != nil {
		return nil, err
	}

	buf.WriteString("\n</body>\n</html>\n")

	name := "report"
	if format.FileName != nil {
		name = *format.FileName
	}

	fd, err := models.NewFileData(&buf, name+".html")
	if err != nil {
		return nil, err
	}

	return []models.Data{fd}, nil
}

func (e Exporter) exportLayout(data models.Dataset, format models.Export) ([]models.Data, error) {
	l := layoutpkg.Normalize(*format.Layout)

	if err := layoutpkg.Validate(l, layoutpkg.SortedDatasetKeys(data)); err != nil {
		return nil, fmt.Errorf("html: invalid layout: %w", err)
	}

	doc, err := layoutpkg.Resolve(l, data)
	if err != nil {
		return nil, fmt.Errorf("html: resolve layout: %w", err)
	}

	pages, err := layoutpkg.Pack(doc, e.Geometry(doc.Page))
	if err != nil {
		return nil, fmt.Errorf("html: pack layout: %w", err)
	}

	return e.Render(context.Background(), data, doc, pages, format)
}

func (e Exporter) exportTemplate(data models.Dataset, format models.Export) ([]models.Data, error) {
	grid := newGridState()

	t, err := template.New("html_tmpl").
		Funcs(sprig.FuncMap()).
		Funcs(grid.funcMap()).
		Funcs(chartFuncMap).
		Funcs(text.FuncMap).
		Parse(format.Template.TemplateText)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(&buf, *format.FileName+".html")
	if err != nil {
		return nil, err
	}

	return []models.Data{fd}, nil
}

func renderPages(
	w io.Writer,
	rc *render.RenderContext,
	pages []layoutpkg.Page,
	page models.PageConfig,
) error {
	pad := page.PaddingMM
	if pad <= 0 {
		pad = layoutpkg.DefaultPaddingMM
	}

	pageStyle := fmt.Sprintf(
		"display: flex; flex-direction: column; padding: %dmm;", pad,
	)

	for _, p := range pages {
		fmt.Fprintf(w, `<div class="report-page" style="%s">`+"\n", pageStyle)

		if err := renderPageSection(w, rc, page.Header, "header", "report-header"); err != nil {
			return err
		}

		fmt.Fprintf(
			w,
			`<div class="report-grid" style="grid-template-rows: repeat(%d, minmax(0, 1fr)); height: auto; flex: 1 1 auto; min-height: 0;">`+"\n",
			layoutpkg.GridRows,
		)

		for _, pb := range p.Blocks {
			style := fmt.Sprintf(
				"grid-column: %d / span %d; grid-row: %d / span %d;",
				pb.Pos.X+1, pb.Pos.W, pb.Pos.Y+1, pb.Pos.H,
			)

			fmt.Fprintf(w, `<div class="report-grid-item" style="%s">`, style)

			if err := renderBlock(w, rc, pb); err != nil {
				return err
			}

			if _, err := io.WriteString(w, "</div>\n"); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(w, "</div>\n"); err != nil {
			return err
		}

		if err := renderPageSection(w, rc, page.Footer, "footer", "report-footer"); err != nil {
			return err
		}

		if _, err := io.WriteString(w, "</div>\n"); err != nil {
			return err
		}
	}

	return nil
}

// renderPageSection writes a running header or footer for a page. HTML wins over
// Text; Text is executed as a text/template over the render dataset. A nil or
// empty section writes nothing.
func renderPageSection(
	w io.Writer,
	rc *render.RenderContext,
	sec *models.PageSection,
	tag, class string,
) error {
	if sec == nil {
		return nil
	}

	switch {
	case strings.TrimSpace(sec.HTML) != "":
		fmt.Fprintf(w, `<%s class="%s">`, tag, class)

		if _, err := io.WriteString(w, sec.HTML); err != nil {
			return err
		}
	case strings.TrimSpace(sec.Text) != "":
		t, err := texttemplate.New(class).Funcs(rc.Funcs).Parse(sec.Text)
		if err != nil {
			return fmt.Errorf("html: parse %s: %w", class, err)
		}

		fmt.Fprintf(w, `<%s class="%s">`, tag, class)

		if err := t.Execute(w, rc.Data); err != nil {
			return fmt.Errorf("html: exec %s: %w", class, err)
		}
	default:
		return nil
	}

	fmt.Fprintf(w, "</%s>\n", tag)

	return nil
}

func docData(doc layoutpkg.ResolvedDocument) models.Dataset {
	data := make(models.Dataset, len(doc.Blocks))
	for _, rb := range doc.Blocks {
		if rb.Block.Dataset != "" {
			data[rb.Block.Dataset] = rb.Rows
		}
	}

	return data
}

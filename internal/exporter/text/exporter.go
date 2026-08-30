package text

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	texttemplate "text/template"

	"support_bot/internal/exporter/render"
	"support_bot/internal/models"

	layoutpkg "support_bot/internal/layout"
)

type Exporter struct{}

// Format implements render.Renderer.
func (e Exporter) Format() string { return models.ReportFormatText }

// Geometry implements render.Renderer: text output is a single unbounded page,
// no pagination.
func (e Exporter) Geometry(_ models.PageConfig) layoutpkg.PageGeometry {
	return layoutpkg.PageGeometry{Rows: 0}
}

// Export renders one export. A declarative layout — or a legacy bare Template
// converted to a single text block — flows through the shared
// Resolve/Pack/Render pipeline.
func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	if format.Layout == nil &&
		(format.Template == nil || format.Template.TemplateText == "") {
		return nil, errors.New("text export: a layout or a non-empty template is required")
	}

	l := layoutpkg.LayoutFromExport(format)

	if err := layoutpkg.Validate(l, layoutpkg.SortedDatasetKeys(data)); err != nil {
		return nil, fmt.Errorf("text: invalid layout: %w", err)
	}

	doc, err := layoutpkg.Resolve(l, data)
	if err != nil {
		return nil, fmt.Errorf("text: resolve layout: %w", err)
	}

	pages, err := layoutpkg.Pack(doc, e.Geometry(doc.Page))
	if err != nil {
		return nil, fmt.Errorf("text: pack layout: %w", err)
	}

	return e.Render(context.Background(), data, doc, pages, format)
}

// Render implements render.Renderer: it executes every text block's body as a
// text/template over data and returns one text file per block.
func (e Exporter) Render(
	_ context.Context,
	data models.Dataset,
	_ layoutpkg.ResolvedDocument,
	pages []layoutpkg.Page,
	format models.Export,
) ([]models.Data, error) {
	fm := render.FuncMap()
	maps.Copy(fm, textMap)

	dataType := "text"
	if format.Template != nil && format.Template.Type != "" {
		dataType = format.Template.Type
	}

	var outputs []models.Data

	for _, page := range pages {
		for _, pb := range page.Blocks {
			if pb.Block.Type != models.BlockTypeText || pb.Block.Text == nil {
				continue
			}

			t, err := texttemplate.New(pb.Block.ID).Funcs(fm).Parse(pb.Block.Text.Text)
			if err != nil {
				return nil, fmt.Errorf("text: parse block %q: %w", pb.Block.ID, err)
			}

			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err != nil {
				return nil, fmt.Errorf("text: render block %q: %w", pb.Block.ID, err)
			}

			outputs = append(outputs, models.NewTextData(&buf, dataType))
		}
	}

	if len(outputs) == 0 {
		return nil, errors.New("text: layout has no text blocks")
	}

	return outputs, nil
}

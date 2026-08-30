package csv

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"

	"support_bot/internal/models"
	"support_bot/internal/pkg"

	layoutpkg "support_bot/internal/layout"
)

type Exporter struct{}

// Format implements render.Renderer.
func (e Exporter) Format() string { return models.ReportFormatCsv }

// Geometry implements render.Renderer: CSV output is a single unbounded page,
// no pagination or table splitting.
func (e Exporter) Geometry(_ models.PageConfig) layoutpkg.PageGeometry {
	return layoutpkg.PageGeometry{Rows: 0}
}

// Export renders one export. A declarative layout (or a legacy Order converted
// to one) flows through the shared Resolve/Pack/Render pipeline and yields one
// CSV file per table block. With neither a layout nor an Order the legacy
// "one file per dataset key" behaviour is kept.
func (e Exporter) Export(data models.Dataset, export models.Export) ([]models.Data, error) {
	if export.Layout == nil && len(export.Order) == 0 {
		return e.exportLegacy(data, export)
	}

	l := layoutpkg.LayoutFromExport(export)

	if err := layoutpkg.Validate(l, layoutpkg.SortedDatasetKeys(data)); err != nil {
		return nil, fmt.Errorf("csv: invalid layout: %w", err)
	}

	doc, err := layoutpkg.Resolve(l, data)
	if err != nil {
		return nil, fmt.Errorf("csv: resolve layout: %w", err)
	}

	pages, err := layoutpkg.Pack(doc, e.Geometry(doc.Page))
	if err != nil {
		return nil, fmt.Errorf("csv: pack layout: %w", err)
	}

	return e.Render(context.Background(), data, doc, pages, export)
}

// Render implements render.Renderer: one CSV file per resolved table block, in
// the column order declared on the block.
func (e Exporter) Render(
	_ context.Context,
	_ models.Dataset,
	_ layoutpkg.ResolvedDocument,
	pages []layoutpkg.Page,
	export models.Export,
) ([]models.Data, error) {
	name := "report"
	if export.FileName != nil {
		name = *export.FileName
	}

	var outputs []models.Data

	for _, page := range pages {
		for _, pb := range page.Blocks {
			if pb.Block.Type != models.BlockTypeTable || pb.Block.Table == nil {
				continue
			}

			cols := pb.Block.Table.Columns
			end := min(pb.RowOffset+pb.RowCount, len(pb.Rows))
			win := pb.Rows[pb.RowOffset:end]

			matrix := make([][]string, 0, len(win)+1)

			header := make([]string, len(cols))
			for i, c := range cols {
				header[i] = c.Field
			}

			matrix = append(matrix, header)

			for _, row := range win {
				rec := make([]string, len(cols))
				for i, c := range cols {
					rec[i] = layoutpkg.FormatValue(row[c.Field], c.Format)
				}

				matrix = append(matrix, rec)
			}

			buf, err := writeStringMatrix(matrix)
			if err != nil {
				return nil, fmt.Errorf("csv: block %q: %w", pb.Block.ID, err)
			}

			dt, err := models.NewFileData(buf, name+"_"+pb.Block.ID+".csv")
			if err != nil {
				return nil, fmt.Errorf("csv: block %q: %w", pb.Block.ID, err)
			}

			outputs = append(outputs, dt)
		}
	}

	if len(outputs) == 0 {
		return nil, errors.New("csv: layout has no table blocks")
	}

	return outputs, nil
}

func (e Exporter) exportLegacy(data models.Dataset, export models.Export) ([]models.Data, error) {
	if err := validateFormat(export); err != nil {
		return nil, fmt.Errorf("validation format error: %w", err)
	}

	var (
		fd  []models.Data
		err error
	)

	for k, v := range data {
		var ordering []string
		if o, ok := export.Order[k]; ok {
			ordering = o
		}

		buf := writeCsv(pkg.ConvertSortedRows(v, ordering))

		dt, eErr := models.NewFileData(buf, *export.FileName+"_"+k+".csv")
		if eErr != nil {
			err = errors.Join(err, eErr)

			continue
		}

		fd = append(fd, dt)
	}

	if err != nil {
		return nil, err
	}

	return fd, nil
}

func validateFormat(format models.Export) error {
	var err error

	if format.FileName == nil || *format.FileName == "" {
		err = errors.Join(err, errors.New("format file name must not be empty"))
	}

	return err
}

// writeStringMatrix writes a fully pre-formatted string matrix (header + rows)
// as CSV. Unlike writeCsv it never stringifies nil as "<nil>".
func writeStringMatrix(rows [][]string) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	w := csv.NewWriter(&buf)
	if err := w.WriteAll(rows); err != nil {
		return nil, err
	}

	return &buf, nil
}

func writeCsv(data [][]any) *bytes.Buffer {
	if len(data) == 0 {
		return nil
	}

	var buf bytes.Buffer

	r := csv.NewWriter(&buf)

	if dt, ok := any(data).([][]string); ok {
		r.WriteAll(dt)

		return &buf
	}

	var rd [][]string

	for _, row := range data {
		outRow := make([]string, len(row))

		for i, v := range row {
			outRow[i] = fmt.Sprint(v)
		}

		rd = append(rd, outRow)
	}

	r.WriteAll(rd)

	return &buf
}

package csv

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"support_bot/internal/models"
	"support_bot/internal/pkg"
)

type Exporter struct{}

func (e Exporter) Export(data models.Dataset, export models.Export) ([]models.Data, error) {
	// Если есть Layout — используем его для рендеринга табличных блоков
	if export.Layout != nil {
		return e.exportFromLayout(data, *export.Layout, export)
	}

	// Legacy режим
	var fd []models.Data

	if err := validateFormat(export); err != nil {
		return nil, fmt.Errorf("validation format error: %w", err)
	}

	var err error

	for k, v := range data {
		var ordering []string

		if o, ok := export.Order[k]; ok {
			ordering = o
		}

		cBuf := pkg.ConvertSortedRows(v, ordering)

		buf := writeCsv(cBuf)

		dt, eErr := models.NewFileData(buf, *export.FileName+"_"+k+".csv")
		if eErr != nil {
			err = errors.Join(err, eErr)

			continue
		}

		fd = append(fd, dt)
	}

	return fd, nil
}

func (e Exporter) exportFromLayout(data models.Dataset, layout models.Layout, export models.Export) ([]models.Data, error) {
	var outputs []models.Data

	for _, block := range layout.Blocks {
		if block.Type != models.BlockTypeTable {
			continue
		}

		if block.Table == nil || block.Dataset == "" {
			continue
		}

		rows, ok := data[block.Dataset]
		if !ok || len(rows) == 0 {
			continue
		}

		// Формируем порядок колонок из блока
		var ordering []string
		for _, col := range block.Table.Columns {
			ordering = append(ordering, col.Field)
		}

		matrix := pkg.ConvertSortedRows(rows, ordering)
		buf := writeCsv(matrix)

		filename := *export.FileName + "_" + block.ID + ".csv"
		dt, err := models.NewFileData(buf, filename)
		if err != nil {
			return nil, fmt.Errorf("create file for block %q: %w", block.ID, err)
		}

		outputs = append(outputs, dt)
	}

	if len(outputs) == 0 {
		return nil, errors.New("no table blocks found in layout")
	}

	return outputs, nil
}

func validateFormat(format models.Export) error {
	var err error

	if format.FileName == nil || *format.FileName == "" {
		err = errors.Join(err, fmt.Errorf("format file name must not be empty"))
	}

	return err
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

package csv

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"

	"support_bot/internal/models"
	"support_bot/internal/pkg"
)

type Exporter struct {
	buf   map[string][]map[string]any
	order map[string][]string
	name  string
}

func New(
	data map[string][]map[string]any,
	name string,
	order map[string][]string,
) *Exporter {
	return &Exporter{
		buf:   data,
		order: order,
		name:  name,
	}
}

func (e *Exporter) Export() ([]models.Data, error) {
	var fd []models.Data

	var err error

	for k, v := range e.buf {
		var ordering []string

		if o, ok := e.order[k]; ok {
			ordering = o
		}

		cBuf := pkg.ConvertSortedRows(v, ordering)

		buf, wErr := writeCsv(cBuf)
		if wErr != nil {
			err = errors.Join(err, wErr)

			continue
		}

		dt, eErr := models.NewFileData(buf, e.name+"_"+k+".csv")
		if eErr != nil {
			err = errors.Join(err, eErr)

			continue
		}

		fd = append(fd, dt)
	}

	return fd, nil
}

func writeCsv(data [][]any) (*bytes.Buffer, error) {
	if len(data) == 0 {
		return nil, nil //nolint:nilnil // nil buffer means no data
	}

	var buf bytes.Buffer

	r := csv.NewWriter(&buf)

	if dt, ok := any(data).([][]string); ok {
		if err := r.WriteAll(dt); err != nil {
			return nil, fmt.Errorf("write csv: %w", err)
		}

		return &buf, nil
	}

	var rd [][]string

	for _, row := range data {
		outRow := make([]string, len(row))

		for i, v := range row {
			outRow[i] = fmt.Sprint(v)
		}

		rd = append(rd, outRow)
	}

	if err := r.WriteAll(rd); err != nil {
		return nil, fmt.Errorf("write csv: %w", err)
	}

	return &buf, nil
}

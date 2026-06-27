package csv

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"support_bot/internal/pkg"

	models "support_bot/internal/models/report"
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
	fd := []models.Data{}

	var err error

	for k, v := range e.buf {
		var ordering []string

		if o, ok := e.order[k]; ok {
			ordering = o
		}

		cBuf := pkg.ConvertSortedRows(v, ordering)

		buf := writeCsv(cBuf)

		dt, eErr := models.NewFileData(buf, e.name+"_"+k+".csv")
		if eErr != nil {
			err = errors.Join(err, eErr)
			continue
		}
		fd = append(fd, dt)
	}

	return fd, nil
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

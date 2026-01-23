package csv

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"

	models "support_bot/internal/models/report"
	"support_bot/internal/pkg"
)

type Exporter[T models.FileData] struct {
	buf   map[string][]map[string]any
	order map[string][]string
	name  string
}

func New[T models.FileData](
	data map[string][]map[string]any,
	name string,
	order map[string][]string,
) *Exporter[T] {
	return &Exporter[T]{
		buf:   data,
		order: order,
		name:  name,
	}
}

func (e *Exporter[T]) Export() (*T, error) {
	fd := models.NewEmptyFileData()

	var err error

	for k, v := range e.buf {
		var ordering []string

		if o, ok := e.order[k]; ok {
			ordering = o
		}

		cBuf := pkg.ConvertSortedRows(v, ordering)

		buf := writeCsv(cBuf)

		eErr := fd.Extend(buf, e.name+"_"+k+".csv")
		if eErr != nil {
			err = errors.Join(err, eErr)
		}
	}

	return any(fd).(*T), nil
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

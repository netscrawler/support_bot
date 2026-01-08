package csv

import (
	"bytes"
	"encoding/csv"

	"support_bot/internal/models"
)

type Exporter[T models.FileData] struct {
	buf  [][]string
	name string
}

func (e *Exporter[T]) Export() (*T, error) {
	return any(models.NewFileData(writeCsv(e.buf), e.name)).(*T), nil
}

func writeCsv(data [][]string) *bytes.Buffer {
	if len(data) == 0 {
		return nil
	}

	var buf bytes.Buffer

	r := csv.NewWriter(&buf)
	r.WriteAll(data)

	return &buf
}

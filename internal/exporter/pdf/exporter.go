package pdf

import (
	"strings"

	models "support_bot/internal/models/report"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

type Exporter[T models.FileData] struct {
	data models.FileData
	name string
}

func New[T models.FileData](data *models.FileData, name string) *Exporter[T] {
	return &Exporter[T]{
		data: *data,
		name: name,
	}
}

func (e *Exporter[T]) Export() (*T, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	for b := range e.data.Data() {
		page := wkhtmltopdf.NewPageReader(strings.NewReader(b.String()))
		pdfg.AddPage(page)
	}

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(pdfg.Buffer(), e.name+".pdf")
	if err != nil {
		return nil, err
	}

	return any(fd).(*T), nil
}

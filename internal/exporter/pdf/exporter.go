package pdf

import (
	"strings"

	models "support_bot/internal/models/report"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

type Exporter struct {
	data []models.Data
	name string
}

func New(name string, data ...models.Data) *Exporter {
	return &Exporter{
		data: data,
		name: name,
	}
}

func (e *Exporter) Export() (*models.Data, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	for _, b := range e.data {
		page := wkhtmltopdf.NewPageReader(strings.NewReader(b.Data.String()))
		pdfg.AddPage(page)
	}

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(pdfg.Buffer(), e.name+".pdf")
	if err != nil {
		return nil, err
	}

	return &fd, nil
}

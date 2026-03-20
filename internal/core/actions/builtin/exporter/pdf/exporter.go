package pdf

import (
	"bytes"
	models "support_bot/internal/models/report"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

type Exporter struct {
	data *models.FileData
	name string
}

func New(data *models.FileData, name string) *Exporter {
	return &Exporter{
		data: data,
		name: name,
	}
}

func (e *Exporter) Export() (*models.FileData, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	page := wkhtmltopdf.NewPageReader(bytes.NewReader(e.data.File.Bytes()))
	pdfg.AddPage(page)

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(pdfg.Buffer(), e.name+".pdf")
	if err != nil {
		return nil, err
	}

	return fd, nil
}

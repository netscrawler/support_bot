package pdf

import (
	"support_bot/internal/models"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

type Exporter[T models.FileData] struct {
	data models.FileData
	name string
}

func (e *Exporter[T]) Export() (*T, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}
	for b := range e.data.Data() {
		page := wkhtmltopdf.NewPage(b.String())
		pdfg.AddPage(page)
	}

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	return any(models.NewFileData(pdfg.Buffer(), e.name)).(*T), nil
}

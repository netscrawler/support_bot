package pdf

import (
	"bytes"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

type Exporter struct {
	data []*bytes.Buffer
}

func (e *Exporter) Export() (*bytes.Buffer, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}
	for _, p := range e.data {
		page := wkhtmltopdf.NewPage(p.String())
		pdfg.AddPage(page)
	}

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	return pdfg.Buffer(), nil
}

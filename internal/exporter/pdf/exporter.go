package pdf

import (
	"strings"
	"support_bot/internal/models"

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
		page.DisableJavascript.Set(false)
		page.EnablePlugins.Set(true)
		page.EnableForms.Set(true)
		page.JavascriptDelay.Set(2000)
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

package html

import (
	"bytes"
	"html/template"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"

	"github.com/Masterminds/sprig/v3"
)

type Exporter struct{}

func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	grid := newGridState()

	t, err := template.New("html_tmpl").
		Funcs(sprig.FuncMap()).
		Funcs(grid.funcMap()).
		Funcs(chartFuncMap).
		Funcs(text.FuncMap).
		Parse(format.Template.TemplateText)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(&buf, *format.FileName+".html")
	if err != nil {
		return nil, err
	}

	return []models.Data{fd}, nil
}

package html

import (
	"bytes"
	"html/template"
	"maps"

	"github.com/Masterminds/sprig/v3"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"
)

type Exporter struct {
	data     any
	template string
	name     string
}

func New(data any, template string, name string) *Exporter {
	return &Exporter{
		data:     data,
		template: template,
		name:     name,
	}
}

func (e *Exporter) Export() (*models.Data, error) {
	allFuncs := sprig.FuncMap()
	maps.Copy(allFuncs, text.FuncMap)

	t, err := template.New("html_tmpl").
		Funcs(allFuncs).
		Parse(e.template)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, e.data); err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(&buf, e.name+".html")
	if err != nil {
		return nil, err
	}

	return &fd, nil
}

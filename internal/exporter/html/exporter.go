package html

import (
	"bytes"
	"html/template"
	"maps"

	"github.com/Masterminds/sprig/v3"
)

type Exporter struct {
	data     any
	template string
}

func New(data any, template string) *Exporter {
	return &Exporter{
		data:     data,
		template: template,
	}
}

func (e *Exporter) Export() (*bytes.Buffer, error) {
	allFuncs := sprig.FuncMap()
	maps.Copy(allFuncs, funcMap)

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

	return &buf, nil
}

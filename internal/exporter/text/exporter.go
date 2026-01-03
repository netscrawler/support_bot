package text

import (
	"bytes"
	"maps"
	"text/template"

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
	allFuncs := sprig.TxtFuncMap()
	maps.Copy(allFuncs, funcMap)

	t, err := template.New("tpl").
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

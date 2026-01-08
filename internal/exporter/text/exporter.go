package text

import (
	"bytes"
	"maps"
	"text/template"

	"support_bot/internal/models"

	"github.com/Masterminds/sprig/v3"
)

type Exporter[T models.TextData] struct {
	data     any
	template string
}

func New[T models.TextData](data any, template string) *Exporter[T] {
	return &Exporter[T]{
		data:     data,
		template: template,
	}
}

func (e *Exporter[T]) Export() (*T, error) {
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

	return any(models.NewTextData(buf.String())).(*T), nil
}

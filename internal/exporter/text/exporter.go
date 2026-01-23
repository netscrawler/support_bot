package text

import (
	"bytes"
	"maps"
	"support_bot/internal/pkg/text"
	"text/template"

	models "support_bot/internal/models/report"

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
	maps.Copy(allFuncs, text.FuncMap)

	t, err := template.New("text_templ").
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

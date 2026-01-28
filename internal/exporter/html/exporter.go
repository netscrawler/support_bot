package html

import (
	"bytes"
	"html/template"
	"maps"
	"support_bot/internal/pkg/text"

	models "support_bot/internal/models/report"

	"github.com/Masterminds/sprig/v3"
)

type Exporter[T models.FileData] struct {
	data     any
	template string
	name     string
}

func New[T models.FileData](data any, template string, name string) *Exporter[T] {
	return &Exporter[T]{
		data:     data,
		template: template,
		name:     name,
	}
}

func (e *Exporter[T]) Export() (*T, error) {
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

	return any(fd).(*T), nil
}

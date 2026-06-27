package text

import (
	"bytes"
	"maps"
	"support_bot/internal/pkg/text"
	"text/template"

	models "support_bot/internal/models/report"

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

func (e *Exporter) Export() (*models.Data, error) {
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

	dt := models.NewTextData(&buf)
	return &dt, nil
}

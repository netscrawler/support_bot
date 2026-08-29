package text

import (
	"bytes"
	"fmt"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type Exporter struct{}

func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	if format.Template == nil || format.Template.TemplateText == "" {
		return nil, fmt.Errorf("text export %q: template is required", *format.FileName)
	}

	t, err := template.New("text_templ").
		Funcs(sprig.TxtFuncMap()).
		Funcs(textMap).
		Funcs(text.FuncMap).
		Parse(format.Template.TemplateText)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}

	dt := models.NewTextData(&buf, format.Template.Type)

	return []models.Data{dt}, nil
}

package text

import (
	"bytes"
	"html/template"
	"maps"

	"github.com/Masterminds/sprig/v3"
)

func ExecuteTemplate(templ string, data ...any) (string, error) {
	allFuncs := sprig.TxtFuncMap()
	maps.Copy(allFuncs, FuncMap)

	t, err := template.New("text_templ").
		Funcs(allFuncs).
		Parse(templ)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

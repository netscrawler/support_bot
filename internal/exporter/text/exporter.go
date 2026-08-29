package text

import (
	"bytes"
	"fmt"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"
	texttemplate "text/template"

	"github.com/Masterminds/sprig/v3"
)

type Exporter struct{}

func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	// Если есть Layout — используем его для рендеринга текстовых блоков
	if format.Layout != nil {
		return e.exportFromLayout(data, *format.Layout, format)
	}

	// Legacy режим: работа с Template.TemplateText
	if format.Template == nil || format.Template.TemplateText == "" {
		return nil, fmt.Errorf("text export %q: template is required", *format.FileName)
	}

	t, err := texttemplate.New("text_templ").
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

func (e Exporter) exportFromLayout(data models.Dataset, layout models.Layout, format models.Export) ([]models.Data, error) {
	var outputs []models.Data

	for _, block := range layout.Blocks {
		if block.Type != models.BlockTypeText {
			continue
		}

		if block.Text == nil {
			continue
		}

		t, err := texttemplate.New(block.ID).
			Funcs(sprig.TxtFuncMap()).
			Funcs(textMap).
			Funcs(text.FuncMap).
			Parse(block.Text.Text)
		if err != nil {
			return nil, fmt.Errorf("parse text block %q: %w", block.ID, err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("render text block %q: %w", block.ID, err)
		}

		// Для layout используем тип из формата или дефолтный
		dataType := "text"
		if format.Template != nil && format.Template.Type != "" {
			dataType = format.Template.Type
		}

		dt := models.NewTextData(&buf, dataType)
		outputs = append(outputs, dt)
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no text blocks found in layout")
	}

	return outputs, nil
}

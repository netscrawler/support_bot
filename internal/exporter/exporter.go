package exporter

import (
	"fmt"

	"support_bot/internal/exporter/csv"
	"support_bot/internal/exporter/html"
	"support_bot/internal/exporter/pdf"
	"support_bot/internal/exporter/png"
	"support_bot/internal/exporter/text"
	"support_bot/internal/exporter/xlsx"
	models "support_bot/internal/models/report"
)

var (
	_ Exporter[*models.TextData]  = (*text.Exporter[models.TextData])(nil)
	_ Exporter[*models.FileData]  = (*xlsx.Exporter[models.FileData])(nil)
	_ Exporter[*models.FileData]  = (*csv.Exporter[models.FileData])(nil)
	_ Exporter[*models.FileData]  = (*html.Exporter[models.FileData])(nil)
	_ Exporter[*models.FileData]  = (*pdf.Exporter[models.FileData])(nil)
	_ Exporter[*models.ImageData] = (*png.Exporter[models.ImageData])(nil)
)

type Exporter[T models.ReportData] interface {
	Export() (T, error)
}

type ExportStrategy struct{}

func Export(
	data map[string][]map[string]any,
	exp models.Export,
) (models.ReportData, error) {
	switch exp.Format {
	case models.ReportFormatCsv:
		r, err := csv.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		return r, nil
	case models.ReportFormatXlsx:
		r, err := xlsx.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		return r, nil
	case models.ReportFormatPng:
		r, err := png.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		return r, nil
	case models.ReportFormatHTML:
		r, err := html.New(data, exp.Template.TemplateText, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		return r, nil

	case models.ReportFormatPdf:
		rh, err := html.New(data, exp.Template.TemplateText, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		r, err := pdf.New(rh, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		return r, nil
	case models.ReportFormatText:
		r, err := text.New(data, exp.Template.TemplateText).Export()
		if err != nil {
			return nil, err
		}

		return r, nil
	default:
		return nil, fmt.Errorf("undefined format: %s", exp.Format)
	}
}

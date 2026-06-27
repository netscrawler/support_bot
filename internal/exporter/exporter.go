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

func Export(
	data map[string][]map[string]any,
	exp models.Export,
) ([]models.Data, error) {
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

		return []models.Data{*r}, nil
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

		return []models.Data{*r}, nil

	case models.ReportFormatPdf:
		rh, err := html.New(data, exp.Template.TemplateText, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		r, err := pdf.New(*exp.FileName, []models.Data{*rh}...).Export()
		if err != nil {
			return nil, err
		}

		return []models.Data{*r}, nil
	case models.ReportFormatText:
		r, err := text.New(data, exp.Template.TemplateText).Export()
		if err != nil {
			return nil, err
		}

		return []models.Data{*r}, nil
	default:
		return nil, fmt.Errorf("undefined format: %s", exp.Format)
	}
}

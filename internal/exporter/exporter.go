package exporter

import (
	"fmt"

	"support_bot/internal/exporter/csv"
	"support_bot/internal/exporter/html"
	"support_bot/internal/exporter/pdf"
	"support_bot/internal/exporter/png"
	"support_bot/internal/exporter/text"
	"support_bot/internal/exporter/xlsx"
	models2 "support_bot/internal/models"
)

func Export(
	data map[string][]map[string]any,
	exp models2.Export,
) ([]models2.Data, error) {
	switch exp.Format {
	case models2.ReportFormatCsv:
		r, err := csv.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		return r, nil
	case models2.ReportFormatXlsx:
		r, err := xlsx.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		return []models2.Data{*r}, nil
	case models2.ReportFormatPng:
		r, err := png.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		return r, nil
	case models2.ReportFormatHTML:
		r, err := html.New(data, exp.Template.TemplateText, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		return []models2.Data{*r}, nil

	case models2.ReportFormatPdf:
		rh, err := html.New(data, exp.Template.TemplateText, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		r, err := pdf.New(*exp.FileName, []models2.Data{*rh}...).Export()
		if err != nil {
			return nil, err
		}

		return []models2.Data{*r}, nil
	case models2.ReportFormatText:
		r, err := text.New(data, exp.Template.TemplateText).Export()
		if err != nil {
			return nil, err
		}

		return []models2.Data{*r}, nil
	default:
		return nil, fmt.Errorf("undefined format: %s", exp.Format)
	}
}

package exporter

import (
	"fmt"
	"support_bot/internal/core/actions/builtin/exporter/csv"
	"support_bot/internal/core/actions/builtin/exporter/html"
	"support_bot/internal/core/actions/builtin/exporter/pdf"
	"support_bot/internal/core/actions/builtin/exporter/png"
	"support_bot/internal/core/actions/builtin/exporter/text"
	"support_bot/internal/core/actions/builtin/exporter/xlsx"
	models "support_bot/internal/models/report"
)

func Export(
	data map[string][]map[string]any,
	exp models.Export,
) ([]*models.ExportedReport, error) {
	var rd []*models.ExportedReport
	switch exp.Format {
	case models.ReportFormatCsv:
		r, err := csv.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}
		for _, record := range r {
			er, err := record.Export()
			if err != nil {
				return nil, err
			}
			rd = append(rd, er)

		}

	case models.ReportFormatXlsx:
		r, err := xlsx.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		er, err := r.Export()
		if err != nil {
			return nil, err
		}
		rd = append(rd, er)

	case models.ReportFormatPng:
		r, err := png.New(data, *exp.FileName, exp.Order).Export()
		if err != nil {
			return nil, err
		}

		for _, record := range r {
			er, err := record.Export()
			if err != nil {
				return nil, err
			}
			rd = append(rd, er)

		}

	case models.ReportFormatHTML:
		r, err := html.New(data, exp.Template.TemplateText, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}
		er, err := r.Export()
		if err != nil {
			return nil, err
		}
		rd = append(rd, er)

	case models.ReportFormatPdf:
		rh, err := html.New(data, exp.Template.TemplateText, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		r, err := pdf.New(rh, *exp.FileName).Export()
		if err != nil {
			return nil, err
		}

		er, err := r.Export()
		if err != nil {
			return nil, err
		}
		rd = append(rd, er)

	case models.ReportFormatText:
		r, err := text.New(data, exp.Template.TemplateText).Export()
		if err != nil {
			return nil, err
		}

		er, err := r.Export()
		if err != nil {
			return nil, err
		}
		rd = append(rd, er)
	default:
		return nil, fmt.Errorf("undefined format: %s", exp.Format)
	}

	return rd, nil
}

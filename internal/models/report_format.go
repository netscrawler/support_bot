package models

type reportFormat = string

const (
	ReportFormatText reportFormat = "text"
	ReportFormatHTML reportFormat = "html"
	ReportFormatPng  reportFormat = "png"
	ReportFormatPdf  reportFormat = "pdf"
	ReportFormatCsv  reportFormat = "csv"
	ReportFormatXlsx reportFormat = "xlsx"
)

var _ = map[string]reportFormat{
	"text": ReportFormatText,
	"html": ReportFormatHTML,
	"png":  ReportFormatPng,
	"pdf":  ReportFormatPdf,
	"csv":  ReportFormatCsv,
	"xlsx": ReportFormatXlsx,
}

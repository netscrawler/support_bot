package models

type ReportFormat = string

const (
	ReportFormatText ReportFormat = "text"
	ReportFormatHTML ReportFormat = "html"
	ReportFormatPng  ReportFormat = "png"
	ReportFormatPdf  ReportFormat = "pdf"
	ReportFormatCsv  ReportFormat = "csv"
	ReportFormatXlsx ReportFormat = "xlsx"
)

var FormatMap = map[string]ReportFormat{
	"text": ReportFormatText,
	"html": ReportFormatHTML,
	"png":  ReportFormatPng,
	"pdf":  ReportFormatPdf,
	"csv":  ReportFormatCsv,
	"xlsx": ReportFormatXlsx,
}

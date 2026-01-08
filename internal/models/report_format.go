package models

type ReportFormat = string

const (
	NotifyFormatText ReportFormat = "text"
	NotifyFormatHTML ReportFormat = "html"
	NotifyFormatPng  ReportFormat = "png"
	NotifyFormatPdf  ReportFormat = "pdf"
	NotifyFormatCsv  ReportFormat = "csv"
	NotifyFormatXlsx ReportFormat = "xlsx"
)

var FormatMap = map[string]ReportFormat{
	"text": NotifyFormatText,
	"html": NotifyFormatHTML,
	"png":  NotifyFormatPng,
	"pdf":  NotifyFormatPdf,
	"csv":  NotifyFormatCsv,
	"xlsx": NotifyFormatXlsx,
}

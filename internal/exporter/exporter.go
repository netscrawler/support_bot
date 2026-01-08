package exporter

import (
	"support_bot/internal/exporter/csv"
	"support_bot/internal/exporter/html"
	"support_bot/internal/exporter/pdf"
	"support_bot/internal/exporter/png"
	"support_bot/internal/exporter/text"
	"support_bot/internal/exporter/xlsx"
	"support_bot/internal/models"
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

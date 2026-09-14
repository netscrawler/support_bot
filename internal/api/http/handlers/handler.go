package handlers

import (
	"log/slog"
	"support_bot/internal/models"
)

type ReportProvider interface {
	GenerateReport(reportID string) (models.Data, error)
}

type Handler struct {
	log *slog.Logger

	rp ReportProvider
}

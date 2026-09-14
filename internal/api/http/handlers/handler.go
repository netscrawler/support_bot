package handlers

import (
	"context"
	"log/slog"
	"support_bot/internal/models"
)

type ReportProvider interface {
	GenerateReport(ctx context.Context, reportID string) (models.Data, error)
}

type Handler struct {
	log *slog.Logger

	rp ReportProvider
}

func NewHandler(rp ReportProvider, log *slog.Logger) *Handler {
	return &Handler{
		rp:  rp,
		log: log.With(slog.Any("module", "http_handler")),
	}
}

package service

import (
	"log/slog"

	"support_bot/internal/sheduler"
)

type Report struct {
	*sheduler.SheduleAPI
	log *slog.Logger
}

func NewReportService(shd *sheduler.SheduleAPI, log *slog.Logger) *Report {
	l := log.With(slog.Any("module", "tg_bot.service.report"))
	return &Report{shd, l}
}

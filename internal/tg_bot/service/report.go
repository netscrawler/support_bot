package service

import (
	"log/slog"
	"support_bot/internal/core"
)

type Report struct {
	*core.SheduleAPI

	log *slog.Logger
}

func NewReportService(shd *core.SheduleAPI, log *slog.Logger) *Report {
	l := log.With(slog.Any("module", "tg_bot.service.report"))

	return &Report{shd, l}
}

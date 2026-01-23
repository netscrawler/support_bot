package repository

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type ReportRepository struct {
	_ *sqlx.DB
	_ *slog.Logger
}

func NewReportRepository(db *sqlx.DB, log *slog.Logger) *ChatRepository {
	l := log.With(slog.Any("module", "tg_bot.repository.report"))

	return &ChatRepository{db: db, log: l}
}

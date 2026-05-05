package di

import (
	"log/slog"

	"support_bot/internal/app/httpapp"
	"support_bot/internal/app/report"
	"support_bot/internal/config"
	"support_bot/internal/core"
	"support_bot/internal/delivery"
	"support_bot/internal/postgres"
	tgbot "support_bot/internal/tg_bot"

	"support_bot/internal/app/bot"

	httpSrv "support_bot/internal/http"

	"gopkg.in/telebot.v4"
)

type container struct {
	bot    *telebot.Bot
	router *tgbot.Router

	shed *core.SheduleAPI

	srv *httpSrv.Server

	sheduleC  chan string
	eventC    chan string
	sheduler  *core.Sheduler
	event     *core.EventCreator
	orch      *core.Orchestrator
	generator *core.Generator
	delivery  *delivery.SenderStrategy

	log *slog.Logger

	storage *postgres.DB
	cfg     *config.Config
	report  *report.App
	tgBot   *bot.Bot
	http    *httpapp.App
}

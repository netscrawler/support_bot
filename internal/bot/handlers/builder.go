package handlers

import (
	"support_bot/internal/bot/middlewares"
	"support_bot/internal/service"

	"gopkg.in/telebot.v4"
)

type serviceBuilder interface {
	Build() (*service.User, *service.Chat, *service.Notify)
}

type HandlerBuilder struct {
	sb  serviceBuilder
	bot *telebot.Bot
}

func NewHB(bot *telebot.Bot, sb serviceBuilder) *HandlerBuilder {
	return &HandlerBuilder{
		sb:  sb,
		bot: bot,
	}
}

func (hb *HandlerBuilder) Build() (*AdminHandler, *UserHandler, *TextHandler, *middlewares.Mw) {
	state := NewState()
	uService, cService, nService := hb.sb.Build()
	aHl := NewAdminHandler(
		hb.bot,
		uService,
		cService,
		nService,
		state,
	)
	uHl := NewUserHandler(
		hb.bot,
		cService,
		uService,
		state,
		nService,
	)
	tHL := NewTextHandler(
		aHl,
		uHl,
		state,
	)
	mw := middlewares.NewMw(uService)

	return aHl, uHl, tHL, mw
}

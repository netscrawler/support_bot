package bot

import (
	"support_bot/internal/bot/handlers"
	"support_bot/internal/bot/menu"
	"support_bot/internal/bot/middlewares"

	"gopkg.in/telebot.v4"
)

type handlerBuilder interface {
	Build() (*handlers.AdminHandler, *handlers.UserHandler, *handlers.TextHandler, *middlewares.Mw)
}

type Router struct {
	bot     *telebot.Bot
	adminHl *handlers.AdminHandler
	textHl  *handlers.TextHandler
	userHl  *handlers.UserHandler
	mw      *middlewares.Mw
}

func NewRouter(
	bot *telebot.Bot,
	hb handlerBuilder,
) *Router {
	aHl, uHl, tHl, mw := hb.Build()

	return &Router{
		bot:     bot,
		adminHl: aHl,
		userHl:  uHl,
		textHl:  tHl,
		mw:      mw,
	}
}

func (r *Router) Setup() {
	register := r.bot.Group()
	register.Handle(menu.RegisterCommand, r.userHl.RegisterUser)

	text := r.bot.Group()
	text.Handle(telebot.OnText, r.textHl.ProcessTextInput, r.mw.TextAuthMiddleware)

	userOnly := r.bot.Group()

	userOnly.Use(r.mw.UserAuthMiddleware)

	userOnly.Handle(menu.UserStart, r.userHl.StartUser)
	userOnly.Handle(&menu.SendNotifyUser, r.userHl.SendNotification)

	userOnly.Handle(
		&telebot.InlineButton{Unique: "confirm_user_notification"},
		r.userHl.ConfirmSendNotification,
	)
	userOnly.Handle(
		&telebot.InlineButton{Unique: "cancel_user_notification"},
		r.userHl.CancelSendNotification,
	)

	adminOnly := r.bot.Group()
	adminOnly.Use(r.mw.AdminAuthMiddleware)
	adminOnly.Handle(menu.StartCommand, r.adminHl.StartAdmin)
	adminOnly.Handle(&menu.ManageUsers, r.adminHl.ManageUsers)
	adminOnly.Handle(&menu.ManageChats, r.adminHl.ManageChats)
	adminOnly.Handle(&menu.ListUser, r.adminHl.ListUsers)
	adminOnly.Handle(&menu.AddUser, r.adminHl.AddUser)
	adminOnly.Handle(&menu.RemoveUser, r.adminHl.RemoveUser)
	adminOnly.Handle(&menu.ListChats, r.adminHl.ListChats)
	adminOnly.Handle(menu.AddChat, r.adminHl.ProcessAddChat)
	adminOnly.Handle(&menu.RemoveChat, r.adminHl.RemoveChat)
	adminOnly.Handle(&menu.Back, r.adminHl.StartAdmin)
	adminOnly.Handle(&menu.SendNotifyAdmin, r.adminHl.SendNotification)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "confirm_notification"},
		r.adminHl.ConfirmSendNotification,
	)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "cancel_notification"},
		r.adminHl.CancelSendNotification,
	)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "add_admin"},
		r.adminHl.AddUserWithAdminRole,
	)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "add_user"},
		r.adminHl.AddUserWithUserRole,
	)
}

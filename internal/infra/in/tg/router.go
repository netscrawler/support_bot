package bot

import (
	"log/slog"
	"support_bot/internal/infra/in/tg/handlers"
	"support_bot/internal/infra/in/tg/menu"
	"support_bot/internal/infra/in/tg/middlewares"

	"gopkg.in/telebot.v4"
	telemw "gopkg.in/telebot.v4/middleware"
)

type Router struct {
	bot     *telebot.Bot
	adminHl *handlers.AdminHandler
	textHl  *handlers.TextHandler
	userHl  *handlers.UserHandler
	mw      *middlewares.Mw
}

func NewRouter(
	bot *telebot.Bot,
	admin *handlers.AdminHandler,
	user *handlers.UserHandler,
	text *handlers.TextHandler,
	mw *middlewares.Mw,
) *Router {
	return &Router{
		bot:     bot,
		adminHl: admin,
		userHl:  user,
		textHl:  text,
		mw:      mw,
	}
}

func (r *Router) Setup() {
	r.bot.Use(telemw.Recover(func(err error, c telebot.Context) {
		l := slog.Default()
		l.Error("recovered from panic", slog.Any("error", err))
	}))
	register := r.bot.Group()
	register.Handle(menu.RegisterCommand, r.userHl.RegisterUser)

	text := r.bot.Group()
	text.Handle(telebot.OnText, r.textHl.ProcessTextInput, r.mw.UserAuthMiddleware)

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
	adminOnly.Handle(menu.InfoCommand, r.adminHl.ProcessInfoCommand)
	adminOnly.Handle(&menu.ManageUsers, r.adminHl.ManageUsers)
	adminOnly.Handle(&menu.ManageChats, r.adminHl.ManageChats)
	adminOnly.Handle(&menu.ListUser, r.adminHl.ListUsers)
	adminOnly.Handle(&menu.AddUser, r.adminHl.AddUser)
	adminOnly.Handle(&menu.RemoveUser, r.adminHl.RemoveUser)
	adminOnly.Handle(&menu.ListChats, r.adminHl.ListChats)
	adminOnly.Handle(menu.AddChat, r.adminHl.ProcessAddChat)
	adminOnly.Handle(menu.AddActiveChat, r.adminHl.ProcessAddActiveChat)
	adminOnly.Handle(&menu.RemoveChat, r.adminHl.RemoveChat)
	adminOnly.Handle(&menu.Back, r.adminHl.StartAdmin)
	adminOnly.Handle(&menu.SendNotifyAdmin, r.adminHl.SendNotification)
	adminOnly.Handle(&menu.StartCron, r.adminHl.StartCronJobs)
	adminOnly.Handle(&menu.ManageCron, r.adminHl.ManageCron)
	adminOnly.Handle(&menu.StopCron, r.adminHl.StopCronJobs)
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

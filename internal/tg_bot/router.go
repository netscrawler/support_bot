package bot

import (
	"log/slog"
	"runtime/debug"
	"support_bot/internal/tg_bot/handlers"
	"support_bot/internal/tg_bot/menu"
	"support_bot/internal/tg_bot/middlewares"

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
		l.Error("recovered from panic", slog.Any("error", err), slog.String("stack", string(debug.Stack())))
	}))
	register := r.bot.Group()
	register.Handle(menu.RegisterCommand, r.userHl.RegisterUser)

	text := r.bot.Group()
	text.Handle(telebot.OnText, r.textHl.ProcessTextInput, r.mw.UserAuthMiddleware)

	userOnly := r.bot.Group()

	userOnly.Use(r.mw.UserAuthMiddleware)

	userOnly.Handle(menu.UserStart, r.userHl.StartUser)
	userOnly.Handle(&menu.LoadAndShowReportUser, r.userHl.LoadReports)
	userOnly.Handle(&telebot.InlineButton{Unique: "back_report_list"}, r.userHl.LoadReportsPage)
	userOnly.Handle(&telebot.InlineButton{Unique: "next_report_list"}, r.userHl.LoadReportsPage)
	userOnly.Handle(&telebot.InlineButton{Unique: "_"}, r.userHl.IgnoreReportPage)
	userOnly.Handle(&telebot.InlineButton{Unique: "report"}, r.userHl.GenerateSelectedReport)

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
	adminOnly.Handle(&menu.StartCron, r.adminHl.StartCronJobs)
	adminOnly.Handle(&menu.ManageCron, r.adminHl.ManageCron)
	adminOnly.Handle(&menu.StopCron, r.adminHl.StopCronJobs)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "add_admin"},
		r.adminHl.AddUserWithAdminRole,
	)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "add_user"},
		r.adminHl.AddUserWithUserRole,
	)
	adminOnly.Handle(&telebot.InlineButton{Unique: "back_report_list"}, r.adminHl.LoadReportsPage)
	adminOnly.Handle(&telebot.InlineButton{Unique: "next_report_list"}, r.adminHl.LoadReportsPage)
	adminOnly.Handle(&telebot.InlineButton{Unique: "_"}, r.adminHl.IgnoreReportPage)
	adminOnly.Handle(&telebot.InlineButton{Unique: "report"}, r.adminHl.GenerateSelectedReport)
}

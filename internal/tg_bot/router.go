package tgbot

import (
	"context"
	"log/slog"

	"support_bot/internal/tg_bot/handlers"
	"support_bot/internal/tg_bot/middlewares"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

type Router struct {
	bot     *telego.Bot
	handler *th.BotHandler
	adminHl *handlers.AdminHandler
	textHl  *handlers.TextHandler
	userHl  handlers.UserHandler
	mw      *middlewares.Mw
}

func NewRouter(
	bot *telego.Bot,
	tHandler *th.BotHandler,
	adminH *handlers.AdminHandler,
	userH *handlers.UserHandler,
	textH *handlers.TextHandler,
	mw *middlewares.Mw,
) *Router {
	base := tHandler.BaseGroup()
	base.Use(th.PanicRecovery())

	user := base.Group(mw.UserFilter)
	user.HandleMessage(userH.Start, th.CommandEqual("start"))
	user.HandleMessage(userH.Help, th.CommandEqual("help"))
	user.HandleCallbackQuery(
		userH.LoadReports,
		th.CallbackDataEqual("reports_show_list"),
	)
	user.HandleCallbackQuery(userH.LoadReportsPage, th.CallbackDataContains("user_report_page"))
	user.HandleCallbackQuery(userH.IgnoreReportPage, th.CallbackDataContains("ignore"))
	user.HandleCallbackQuery(userH.GenerateSelectedReport, th.CallbackDataContains("report_gen"))
	// Generic back handler to allow returning to previous user menu
	user.HandleCallbackQuery(userH.Back, th.CallbackDataEqual("user_back"))

	admin := base.Group(mw.AdminFilter)

	admin.HandleMessage(adminH.Start, th.CommandEqual("admin"))
	admin.HandleCallbackQuery(adminH.IgnoreReportPage, th.CallbackDataContains("ignore"))
	admin.HandleCallbackQuery(adminH.ListReports, th.CallbackDataEqual("admin_reports_show_list"))
	admin.HandleCallbackQuery(adminH.SelectReport, th.CallbackDataContains("admin_report_select"))
	admin.HandleCallbackQuery(adminH.LoadReportPage, th.CallbackDataContains("admin_report_page"))
	admin.HandleCallbackQuery(adminH.ResendSelectReport, th.CallbackDataContains("report_resend"))
	admin.HandleCallbackQuery(adminH.GenerateSelectedReport, th.CallbackDataContains("report_get"))
	admin.HandleCallbackQuery(adminH.LoadReportPage, th.CallbackDataContains("back_to_report_list"))
	admin.HandleCallbackQuery(adminH.ManageUsers, th.CallbackDataEqual("manage_users"))
	admin.HandleCallbackQuery(adminH.ListUsers, th.CallbackDataEqual("list_user"))
	admin.HandleCallbackQuery(adminH.AddUser, th.CallbackDataEqual("add_user"))
	admin.HandleCallbackQuery(adminH.DeleteUser, th.CallbackDataContains("delete_user"))
	admin.HandleCallbackQuery(adminH.RemoveUser, th.CallbackDataEqual("remove_user"))
	admin.HandleCallbackQuery(adminH.ShowUser, th.CallbackDataContains("show_user"))
	admin.HandleCallbackQuery(adminH.ListChats, th.CallbackDataEqual("manage_chats"))
	admin.HandleMessage(adminH.AddChat, th.CommandEqual("add"))
	admin.HandleCallbackQuery(adminH.DeleteChat, th.CallbackDataContains("delete_chat"))
	admin.HandleCallbackQuery(
		adminH.ConfirmDeleteChat,
		th.CallbackDataContains("confirm_delete_chat"),
	)
	admin.HandleCallbackQuery(adminH.ShowChat, th.CallbackDataContains("show_chat"))
	admin.HandleCallbackQuery(adminH.ManageCrons, th.CallbackDataEqual("manage_cron"))
	admin.HandleCallbackQuery(adminH.ListCrons, th.CallbackDataEqual("list_cron"))
	admin.HandleCallbackQuery(adminH.StartJobs, th.CallbackDataEqual("start_cron"))
	admin.HandleCallbackQuery(adminH.StopJobs, th.CallbackDataEqual("stop_cron"))
	// Generic back handler to return to admin main menu from any submenu
	admin.HandleCallbackQuery(adminH.Back, th.CallbackDataEqual("back"))

	r := &Router{
		bot:     bot,
		handler: tHandler,
	}

	return r
}

func (r *Router) Start() {
	if r.handler.IsRunning() {
		return
	}

	go func() {
		slog.Debug("start routing")
		err := r.handler.Start()
		slog.Debug("stop routing", slog.Any("err", err))
	}()
}

func (r *Router) Stop(ctx context.Context) error {
	return r.handler.StopWithContext(ctx)
}

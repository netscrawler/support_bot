package menu

import "gopkg.in/telebot.v4"

var (
	AdminMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	UserMenu  = &telebot.ReplyMarkup{ResizeKeyboard: true}
	Selector  = &telebot.ReplyMarkup{}
)

var LoadAndShowReportUser = UserMenu.Text("Отчеты")

var (
	ManageUsers = AdminMenu.Text("👥 Управление пользователями")
	ManageChats = AdminMenu.Text("💬 Управление чатами")
	ManageCron  = AdminMenu.Text("🔄 Управление рассылками")
	StartCron   = AdminMenu.Text("🔄 Перезапустить рассылки")
	StopCron    = AdminMenu.Text("🔄 Выключить рассылку")

	ListUser   = AdminMenu.Text("📋 Список пользователей")
	AddUser    = AdminMenu.Text("➕ Добавить пользователя")
	RemoveUser = AdminMenu.Text("➖ Удалить пользователя")

	ListChats  = AdminMenu.Text("📋 Список чатов")
	RemoveChat = AdminMenu.Text("➖ Удалить чат")

	Back = AdminMenu.Text("🔙 Назад")
)

var (
	StartCommand    = "/admin"
	InfoCommand     = "/info"
	UserStart       = "/start"
	AddChat         = "/add"
	AddActiveChat   = "/sub"
	RegisterCommand = "/register"
)

var MsgHelloReport = `Выберите нужный отчет и он придет в данный чат`

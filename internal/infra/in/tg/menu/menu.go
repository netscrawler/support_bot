package menu

import "gopkg.in/telebot.v4"

var (
	AdminMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	UserMenu  = &telebot.ReplyMarkup{ResizeKeyboard: true}
	Selector  = &telebot.ReplyMarkup{}
)

var (
	SendNotifyAdmin = AdminMenu.Text("📝 Начать рассылку")
	SendNotifyUser  = UserMenu.Text("📝 Сделать рассылку")
)

var (
	ManageUsers   = AdminMenu.Text("👥 Управление пользователями")
	ManageChats   = AdminMenu.Text("💬 Управление чатами")
	RestartCron   = AdminMenu.Text("🔄 Перезапустить крон-задачи")
	DisableNotify = AdminMenu.Text("🔄 Выключить рассылку")
	EnableNotify  = AdminMenu.Text("🔄 Выключить рассылку")

	ListUser   = AdminMenu.Text("📋 Список пользователей")
	AddUser    = AdminMenu.Text("➕ Добавить пользователя")
	RemoveUser = AdminMenu.Text("➖ Удалить пользователя")

	ListChats  = AdminMenu.Text("📋 Список чатов")
	RemoveChat = AdminMenu.Text("➖ Удалить чат")

	ListNotify = AdminMenu.Text("📋 Список уведомлений")

	Back = AdminMenu.Text("🔙 Назад")
)

var (
	StartCommand    = "/admin"
	UserStart       = "/start"
	AddChat         = "/subscribe"
	RegisterCommand = "/register"
)

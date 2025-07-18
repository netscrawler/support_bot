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
	ManageUsers = AdminMenu.Text("👥 Управление пользователями")
	ManageChats = AdminMenu.Text("💬 Управление чатами")

	ListUser   = AdminMenu.Text("📋 Список пользователей")
	AddUser    = AdminMenu.Text("➕ Добавить пользователя")
	RemoveUser = AdminMenu.Text("➖ Удалить пользователя")

	ListChats  = AdminMenu.Text("📋 Список чатов")
	RemoveChat = AdminMenu.Text("➖ Удалить чат")

	Back = AdminMenu.Text("🔙 Назад")
)

var (
	StartCommand    = "/admin"
	UserStart       = "/start"
	AddChat         = "/subscribe"
	RegisterCommand = "/register"
)

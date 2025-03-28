package menu

import "gopkg.in/telebot.v4"

var (
	AdminMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	UserMenu  = &telebot.ReplyMarkup{ResizeKeyboard: true}
	Selector  = &telebot.ReplyMarkup{}
)

var (
	Confirm = Selector.Data("yes", "SendConirm")
	Dismiss = Selector.Data("no", "SendDismiss")
)

var (
	SendNotifyAdmin = AdminMenu.Text("📝 Send Notification")
	SendNotifyUser  = UserMenu.Text("📝 Send Notification")
)

var (
	ManageUsers = AdminMenu.Text("👥 Manage Users")
	ManageChats = AdminMenu.Text("💬 Manage Chats")

	ListUser   = AdminMenu.Text("📋 List Users")
	AddUser    = AdminMenu.Text("➕ Add User")
	RemoveUser = AdminMenu.Text("➖ Remove User")

	ListChats  = AdminMenu.Text("📋 List Chats")
	AddChat    = AdminMenu.Text("➕ Add Chat")
	RemoveChat = AdminMenu.Text("➖ Remove Chat")

	Back = AdminMenu.Text("🔙 Back to Admin Menu")
)

var StartCommand = "/start"

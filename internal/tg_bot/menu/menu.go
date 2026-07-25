package menu

import (
	"github.com/mymmrac/telego"
)

var (
	ManageUsers = telego.InlineKeyboardButton{
		Text:         "👥 Управление пользователями",
		CallbackData: "manage_users",
	}

	ManageChats = telego.InlineKeyboardButton{
		Text:         "💬 Управление чатами",
		CallbackData: "manage_chats",
	}

	ManageCron = telego.InlineKeyboardButton{
		Text:         "🔄 Управление рассылками",
		CallbackData: "manage_cron",
	}

	StartCron = telego.InlineKeyboardButton{
		Text:         "▶️ Запустить рассылки",
		CallbackData: "start_cron",
	}

	StopCron = telego.InlineKeyboardButton{
		Text:         "⏹ Остановить рассылки",
		CallbackData: "stop_cron",
	}

	ListUser = telego.InlineKeyboardButton{
		Text:         "📋 Список пользователей",
		CallbackData: "list_user",
	}

	AddUser = telego.InlineKeyboardButton{
		Text:         "➕ Добавить пользователя",
		CallbackData: "add_user",
	}

	RemoveUser = telego.InlineKeyboardButton{
		Text:         "➖ Удалить пользователя",
		CallbackData: "remove_user",
	}

	ListChats = telego.InlineKeyboardButton{
		Text:         "📋 Список чатов",
		CallbackData: "list_chats",
	}

	RemoveChat = telego.InlineKeyboardButton{
		Text:         "➖ Удалить чат",
		CallbackData: "remove_chat",
	}

	Back = telego.InlineKeyboardButton{
		Text:         "⬅️ Назад",
		CallbackData: "back",
	}

	ShowReports = telego.InlineKeyboardButton{
		Text:         "📊 Выбрать отчёт",
		CallbackData: "reports_show_list",
	}

	ShowReportsAdmin = telego.InlineKeyboardButton{
		Text:         "📊 Отчёты",
		CallbackData: "admin_reports_show_list",
	}
)

var (
	StartCommand    = "/admin"
	InfoCommand     = "/info"
	UserStart       = "/start"
	AddChat         = "/add"
	AddActiveChat   = "/sub"
	RegisterCommand = "/register"
)

var MsgHelloReport = `Выберите отчёт из списка ниже. Я запущу его и отправлю результат в этот чат.`

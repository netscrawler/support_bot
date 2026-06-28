package handlers

const (
	helloAdminRegistration    = "Добро пожаловать! Вы зарегистрированы как администратор"
	ManageUsers               = "Управление пользователями"
	userAddRemove             = "Пожалуйста, отправьте мне username пользователя (@username) в Telegram, которого вы хотите добавить."
	pleaseSendCorrectUsername = "Пожалуйста пришлите username начинающийся с @"
)

const (
	pleaseSendMessage = "Пожалуйста, пришлите мне сообщение, которое вы хотите отправить."
)

const (
	unableSendMessages = "Не удалось отправить уведомление: "

	errDeleteUser             = "Ошибка удаления пользователя: "
	errDeleteUserCauseSuicide = errDeleteUser + "нельзя удалить себя"
)

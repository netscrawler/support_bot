package handlers

const (
	HelloAdminRegistration    = "Добро пожаловать! Вы зарегистрированы как администратор"
	ManageUsers               = "Управление пользователями"
	UserAddRemove             = "Пожалуйста, отправьте мне username пользователя (@username) в Telegram, которого вы хотите добавить."
	PleaseSendCorrectUsername = "Пожалуйста пришлите username начинающийся с @"
)

const (
	PleaseSendMessage = "Пожалуйста, пришлите мне сообщение, которое вы хотите отправить."
)

const (
	SendTimeExpired = "Время на отправку истекло, начните заново"

	UnableSendMessages = "Не удалось отправить уведомление: "

	UnableCauseNotFound = UnableSendMessages + "не нашлось чатов для отправки"
	UnableCauseInternal = UnableSendMessages + "внутренняя ошибка"

	ErrDeleteUser             = "Ошибка удаления пользователя: "
	ErrDeleteUserCauseSuicide = ErrDeleteUser + "нельзя удалить себя"
)

const (
	SendNotifyAborted = "❌ Отправка уведомлений отменена."
)

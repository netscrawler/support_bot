package service

import (
	"fmt"

	"support_bot/internal/models"
)

const (
	addNewChatSuccessTemplate = `Добавлен новый чат в рассылку: %s`
	addNewChatErrorTemplate   = `Ошибка добавления нового чата:
%s
Ошибка: %s`
)
const newUserRegisteredSuccessTemplate = ``

func newAddNewChatSuccessTemplate(chat models.TgChatDTO) string {
	return fmt.Sprintf(addNewChatSuccessTemplate, chat.String())
}

func newAddNewChatErrorTemplate(chat models.TgChatDTO, err error) string {
	return fmt.Sprintf(addNewChatErrorTemplate, chat.String(), err)
}

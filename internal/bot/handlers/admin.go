package handlers

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"support_bot/internal/bot/menu"
	"support_bot/internal/models"
	"support_bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

type AdminHandler struct {
	bot         *tele.Bot
	userService *service.User
	chatService *service.Chat
	state       *State
	notify      *service.Notify
}

func NewAdminHandler(
	bot *tele.Bot,
	userService *service.User,
	chatService *service.Chat,
	notificationService *service.Notify,
	state *State,
) *AdminHandler {
	return &AdminHandler{
		bot:         bot,
		userService: userService,
		chatService: chatService,
		state:       state,
		notify:      notificationService,
	}
}

// StartAdmin handles the start command for admins
func (h *AdminHandler) StartAdmin(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.ManageUsers, menu.ManageChats),
		menu.AdminMenu.Row(menu.SendNotifyAdmin),
	)
	h.state.Set(c.Sender().ID, MenuState)

	return c.Send("Добро пожаловать! Вы зарегистрированы как администратор", menu.AdminMenu)
}

func (h *AdminHandler) SendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, SendNotificationState)
	return c.Send("Пожалуйста, пришлите мне сообщение, которое вы хотите отправить.")
}

func (h *AdminHandler) ProcessSendNotification(c tele.Context) error {
	if h.state.Get(c.Sender().ID) != SendNotificationState {
		return nil
	}

	msg := c.Text()
	if msg == "" {
		return c.Send("Пожалуйста, пришлите мне сообщение, которое вы хотите отправить.")
	}

	// Создаем inline-клавиатуру с кнопками "Confirm" и "Cancel"
	confirmBtn := menu.Selector.Data("✅ Отправить", "confirm_notification", msg)

	menu.Selector.Inline(
		menu.Selector.Row(confirmBtn),
		menu.Selector.Row(menu.Selector.Data("❌ Отменить", "cancel_notification", msg)),
	)

	// Сохраняем состояние ожидания подтверждения
	h.state.Set(c.Sender().ID, ConfirmNotificationState)

	conf := "Вы уверены, что хотите отправить это уведомление?\n\n"
	formated := fmt.Sprintf("%s```%s```", conf, msg)

	// Отправляем сообщение с клавиатурой
	return c.Send(
		formated,
		menu.Selector,
		tele.ModeMarkdownV2,
	)
}

// Confirm sending notification
func (h *AdminHandler) ConfirmSendNotification(c tele.Context) error {
	if h.state.Get(c.Sender().ID) != ConfirmNotificationState {
		return nil
	}

	msg := c.Data()
	num, successfully, witherror, err := h.notify.Broadcast(context.TODO(), h.bot, msg)
	if err != nil {
		return c.Send("Не удалось отправить уведомление: " + err.Error())
	}

	h.state.Set(c.Sender().ID, MenuState)
	formattedMsg := fmt.Sprintf(
		"✅ **Уведомления отправлены**\n\n"+
			"Всего чатов: **%d**\n"+
			"Успешно: **%d**\n"+
			"Не отправленно: **%d**\n\n"+
			"*Note: Пожалуйста, проверьте, есть ли какие-либо особые проблемы в неудачных чатах.*",
		num, successfully, witherror,
	)
	return c.Edit(formattedMsg, tele.ModeMarkdownV2)
}

// Cancel sending notification
func (h *AdminHandler) CancelSendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, MenuState)
	return c.Edit("❌ Отправка уведомлений отменена.")
}

// ManageUsers handles the user management menu
func (h *AdminHandler) ManageUsers(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.AddUser, menu.RemoveUser),
		menu.AdminMenu.Row(menu.ListUser, menu.Back))
	h.state.Set(c.Sender().ID, MenuState)
	return c.Send("Управление пользователями", menu.AdminMenu)
}

// Универсальный обработчик текстовых сообщений
func (h *AdminHandler) ProcessAdminInput(c tele.Context) error {
	userID := c.Sender().ID

	// Проверяем, что юзер в состоянии
	if h.state.Get(userID) == MenuState {
		return nil
	}

	state := h.state.Get(userID)

	switch state {
	case AddUserState:
		return h.ProcessAddUser(c) // Вызываем обработку добавления пользователя
	case RemoveUserState:
		return h.ProcessRemoveUser(c)
	case AddChatState:
		return h.ProcessAddChat(c)
	case RemoveChatState:
		return h.ProcessRemoveChat(c)
	case SendNotificationState:
		return h.ProcessSendNotification(c)
	default:
		return nil // Если нет активного состояния — игнорируем
	}
}

// Мне нужно чтобы после срабатывания этого хендлера бот ждал пока пользователь не напишет в чат ник который нужно добавить
func (h *AdminHandler) AddUser(c tele.Context) error {
	h.state.Set(c.Sender().ID, AddUserState)
	return c.Send(
		"Пожалуйста, отправьте мне username пользователя (@username) в Telegram, которого вы хотите добавить.",
	)
}

// ProcessAddUser processes the username input for adding a user
func (h *AdminHandler) ProcessAddUser(c tele.Context) error {
	userID := c.Sender().ID
	if h.state.Get(userID) != AddUserState {
		return nil
	}

	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send("Пожалуйста пришлите username начинающийся с @")
	}

	username = username[1:]

	if err := h.userService.Create(context.Background(), rand.Int64(), username, "", ""); err != nil {
		return c.Send("Не удалось добавить пользователя: " + err.Error())
	}

	h.state.Set(userID, MenuState) // Сбрасываем состояние
	return c.Send("Пользователь @" + username + " добавлен.")
}

// RemoveUser handles removing a user
func (h *AdminHandler) RemoveUser(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveUserState)
	return c.Send(
		"Пожалуйста, отправьте мне username пользователя (@username) в Telegram, которого вы хотите добавить.",
	)
}

// ProcessRemoveUser processes the username input for removing a user
func (h *AdminHandler) ProcessRemoveUser(c tele.Context) error {
	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send("Пожалуйста пришлите username начинающийся с @")
	}

	// Remove @ and extract the username
	username = username[1:]
	ctx := context.Background()

	// Call service to remove user
	err := h.userService.Delete(ctx, username)
	if err != nil {
		return c.Send("Ошибка удаления пользователя: " + err.Error())
	}
	h.state.Set(c.Sender().ID, MenuState)

	return c.Send("Пользователь @" + username + " успешно удален!")
}

// ListUsers handles listing all users
func (h *AdminHandler) ListUsers(c tele.Context) error {
	users, err := h.userService.GetAll(context.TODO())
	if errors.Is(err, models.ErrNotFound) {
		return c.Send("Пользователи не найдены.")
	}
	if err != nil {
		return c.Send("Ошибка получения пользователей: " + err.Error())
	}

	var response strings.Builder
	response.WriteString("📋 *Список пользователей:*\n\n")

	for i, user := range users {
		response.WriteString(
			fmt.Sprintf("%d. @%s - Role: %s\n", i+1, user.Username, user.Role),
		)
	}

	return c.Send(response.String(), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

// ManageChats handles the chat management menu
func (h *AdminHandler) ManageChats(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.RemoveChat),
		menu.AdminMenu.Row(menu.ListChats, menu.Back))
	return c.Send("Управление чатами", menu.AdminMenu)
}

// ProcessAddChat processes the chat input for adding a chat
func (h *AdminHandler) ProcessAddChat(c tele.Context) error {
	// Проверяем, откуда пришла команда (чат или личка)
	if c.Chat().Type == tele.ChatPrivate {
		return c.Send("Эта команда может использоваться только в чатах")
	}

	chat := &models.Chat{
		ChatID:      c.Chat().ID,
		Title:       c.Chat().Title,
		Type:        string(c.Chat().Type),
		Description: c.Chat().Description,
	}

	err := h.chatService.Add(chat)
	if err != nil {
		return c.Send("Ошибка добавления чата: " + err.Error())
	}

	return c.Send("Чат успешно добавлен")
}

// RemoveChat handles removing a chat
func (h *AdminHandler) RemoveChat(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveChatState)
	return c.Send("Пожалуйста пришлите имя чата (@title) который вы хотите удалить.")
}

// ProcessRemoveChat processes the chat input for removing a chat
func (h *AdminHandler) ProcessRemoveChat(c tele.Context) error {
	chatName := c.Text()
	if !strings.HasPrefix(chatName, "@") {
		return c.Send("Пожалуйста пришлите имя чата начинающиеся с @")
	}

	// Remove @ and extract the username
	chatName = chatName[1:]

	// Call service to remove chat
	err := h.chatService.Remove(chatName)
	if err != nil {
		return c.Send("Ошибка удаления чата: " + err.Error())
	}
	h.state.Set(c.Sender().ID, MenuState)
	return c.Send("Чат @" + chatName + " успешно удален!")
}

// ListChats handles listing all chats
func (h *AdminHandler) ListChats(c tele.Context) error {
	chats, err := h.chatService.GetAll()
	if err != nil {
		return c.Send("Ошибка получения чатов: " + err.Error())
	}

	if len(chats) == 0 {
		return c.Send("Чатов не найдено.")
	}

	var response strings.Builder
	response.WriteString("📋 *Список чатов:*\n\n")

	for i, chat := range chats {
		response.WriteString(
			fmt.Sprintf("%d. @%s\n", i+1, chat.Title),
		)
	}

	return c.Send(response.String(), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

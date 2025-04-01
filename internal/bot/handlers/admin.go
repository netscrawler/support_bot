package handlers

import (
	"context"
	"errors"
	"fmt"
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
	chatNotify  *service.ChatNotify
	userNotify  *service.UserNotify
	state       *State
}

func NewAdminHandler(
	bot *tele.Bot,
	userService *service.User,
	chatService *service.Chat,
	notificationService *service.ChatNotify,
	userNotify *service.UserNotify,
	state *State,
) *AdminHandler {
	return &AdminHandler{
		bot:         bot,
		userService: userService,
		chatService: chatService,
		state:       state,
		chatNotify:  notificationService,
		userNotify:  userNotify,
	}
}

// StartAdmin handles the start command for admins
func (h *AdminHandler) StartAdmin(c tele.Context) error {
	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.ManageUsers, menu.ManageChats),
		menu.AdminMenu.Row(menu.SendNotifyAdmin),
	)
	h.state.Set(c.Sender().ID, MenuState)
	//nolint:errcheck
	c.Delete()

	return c.Send("Добро пожаловать! Вы зарегистрированы как администратор", menu.AdminMenu)
}

func (h *AdminHandler) SendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, SendNotificationState)

	//nolint:errcheck
	c.Delete()

	return c.Send("Пожалуйста, пришлите мне сообщение, которое вы хотите отправить.")
}

func (h *AdminHandler) ProcessSendNotification(c tele.Context) error {
	if h.state.Get(c.Sender().ID) != SendNotificationState {
		return c.Edit("Время на отправку истекло, начните заново")
	}

	msg := c.Text()
	if msg == "" {
		return c.Send("Пожалуйста, пришлите мне сообщение, которое вы хотите отправить.")
	}

	h.state.SetMsgData(c.Sender().ID, msg)
	// Создаем inline-клавиатуру с кнопками "Confirm" и "Cancel"
	confirmBtn := menu.Selector.Data(
		"✅ Отправить",
		"confirm_notification",
	)
	cancelBtn := menu.Selector.Data("❌ Отменить", "cancel_notification")

	menu.Selector.Inline(
		menu.Selector.Row(cancelBtn, confirmBtn),
	)

	// Сохраняем состояние ожидания подтверждения
	h.state.Set(c.Sender().ID, ConfirmNotificationState)

	conf := "Вы уверены, что хотите отправить это уведомление?\n\n"
	formated := fmt.Sprintf("%s```\n%s```", conf, msg)

	// Отправляем сообщение с клавиатурой
	return c.Send(
		formated,
		menu.Selector,
		tele.ModeMarkdownV2,
	)
}

// Confirm sending notification
func (h *AdminHandler) ConfirmSendNotification(c tele.Context) error {
	ctx := context.Background()
	msg, ok := h.state.GetMsgData(c.Sender().ID)
	if h.state.Get(c.Sender().ID) != ConfirmNotificationState || !ok {
		return c.Edit("Время на подтверждение истекло")
	}

	resp, err := h.chatNotify.Broadcast(ctx, h.bot, msg)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Edit("Не удалось отправить уведомление: не нашлось чатов для отправки")
		}
		if errors.Is(err, models.ErrInternal) {
			return c.Edit("Не удалось отправить уведомление: внутренняя ошибка")
		}
		return c.Edit("Не удалось отправить уведомление: " + err.Error())
	}

	userString := fmt.Sprintf("Пользователь @%s разослал уведомление:", c.Sender().Username)
	formString := fmt.Sprintf(
		"%s\n```\n%s```",
		userString, msg,
	)
	//nolint:errcheck
	h.userNotify.SendAdminNotify(ctx, h.bot, formString)
	h.state.Set(c.Sender().ID, MenuState)
	return c.Edit(resp, tele.ModeMarkdownV2)
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
	//nolint:errcheck
	c.Delete()
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

func (h *AdminHandler) AddUser(c tele.Context) error {
	h.state.Set(c.Sender().ID, AddUserState)
	//nolint:errcheck
	c.Delete()
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
	// nolint: errcheck
	c.Delete()

	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send("Пожалуйста пришлите username начинающийся с @")
	}

	username = username[1:]
	confirmBtn := menu.Selector.Data(
		"Admin",
		"add_admin",
		username,
	)
	cancelBtn := menu.Selector.Data("User", "add_user", username)

	menu.Selector.Inline(
		menu.Selector.Row(cancelBtn, confirmBtn),
	)
	// nolint: errcheck
	c.Delete()

	return c.Send("Выберите роль для пользователя @"+username+".", menu.Selector)
}

func (h *AdminHandler) AddUserWithUserRole(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID
	if h.state.Get(userID) != AddUserState {
		return nil
	}

	username := c.Data()

	if err := h.userService.CreateEmpty(ctx, username, false); err != nil {
		return c.Send("Не удалось добавить пользователя: " + err.Error())
	}

	h.state.Set(userID, MenuState) // Сбрасываем состояние
	return c.Edit("Пользователь @" + username + " добавлен.")
}

func (h *AdminHandler) AddUserWithAdminRole(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID
	if h.state.Get(userID) != AddUserState {
		return nil
	}

	username := c.Data()

	if err := h.userService.CreateEmpty(ctx, username, true); err != nil {
		return c.Edit("Не удалось добавить пользователя: " + err.Error())
	}

	h.state.Set(userID, MenuState) // Сбрасываем состояние
	return c.Edit("Администратор @" + username + " добавлен.")
}

// RemoveUser handles removing a user
func (h *AdminHandler) RemoveUser(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveUserState)
	//nolint:errcheck
	c.Delete()
	return c.Send(
		"Пожалуйста, отправьте мне username пользователя (@username) в Telegram, которого вы хотите добавить.",
	)
}

// ProcessRemoveUser processes the username input for removing a user
func (h *AdminHandler) ProcessRemoveUser(c tele.Context) error {
	ctx := context.Background()
	isPrimeReq := false
	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send("Пожалуйста пришлите username начинающийся с @")
	}

	// Remove @ and extract the username
	username = username[1:]
	if username == c.Sender().Username {
		return c.Send("Ошибка удаления пользователя: нельзя удалить себя")
	}

	role, err := h.userService.IsAllowed(ctx, c.Sender().ID)
	if role == models.PrimaryAdminRole {
		isPrimeReq = true
	}
	if err != nil {
		return c.Send("Ошибка удаления пользователя: " + err.Error())
	}

	// Call service to remove user
	err = h.userService.Delete(ctx, username, isPrimeReq)
	if err != nil {
		return c.Send("Ошибка удаления пользователя: " + err.Error())
	}
	h.state.Set(c.Sender().ID, MenuState)

	return c.Send("Пользователь @" + username + " успешно удален!")
}

// ListUsers handles listing all users
func (h *AdminHandler) ListUsers(c tele.Context) error {
	ctx := context.Background()
	users, err := h.userService.GetAll(ctx)
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
	//nolint:errcheck
	c.Delete()

	return c.Send(response.String(), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

// ManageChats handles the chat management menu
func (h *AdminHandler) ManageChats(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.RemoveChat),
		menu.AdminMenu.Row(menu.ListChats, menu.Back))
	//nolint:errcheck
	c.Delete()
	return c.Send("Управление чатами", menu.AdminMenu)
}

// ProcessAddChat processes the chat input for adding a chat
func (h *AdminHandler) ProcessAddChat(c tele.Context) error {
	ctx := context.Background()
	if c.Chat().Type == tele.ChatPrivate {
		return c.Send("Эта команда может использоваться только в чатах")
	}
	// nolint:errcheck, в данном случае не важно смог он удалить сообщение или нет
	c.Delete()
	err := h.chatService.Add(ctx, c.Chat())
	if err != nil {
		// nolint:errcheck, возникновение ошибки не влияет на бизнес логику
		h.userNotify.SendNotify(
			ctx,
			h.bot,
			c.Sender().ID,
			fmt.Sprintf("Ошибка добавления чата: %s : %v", c.Chat().Title, err.Error()),
		)
		return nil
	}

	// nolint:errcheck, возникновение ошибки не влияет на бизнес логику
	h.userNotify.Broadcast(
		ctx,
		h.bot,
		fmt.Sprintf("Добавлен новый чат для рассылки: %s", c.Chat().Title),
	)
	return nil
}

// RemoveChat handles removing a chat
func (h *AdminHandler) RemoveChat(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveChatState)
	//nolint:errcheck
	c.Delete()
	return c.Send("Пожалуйста пришлите имя чата (@title) который вы хотите удалить.")
}

// ProcessRemoveChat processes the chat input for removing a chat
func (h *AdminHandler) ProcessRemoveChat(c tele.Context) error {
	ctx := context.Background()
	chatName := c.Text()
	if !strings.HasPrefix(chatName, "@") {
		return c.Send("Пожалуйста пришлите имя чата начинающиеся с @")
	}

	chatName = chatName[1:]

	err := h.chatService.Remove(ctx, chatName)
	if err != nil {
		return c.Send("Ошибка удаления чата: " + err.Error())
	}
	h.state.Set(c.Sender().ID, MenuState)
	return c.Send("Чат @" + chatName + " успешно удален!")
}

// ListChats handles listing all chats
func (h *AdminHandler) ListChats(c tele.Context) error {
	ctx := context.Background()
	chats, err := h.chatService.GetAll(ctx)
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
	//nolint:errcheck
	c.Delete()

	return c.Send(response.String(), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

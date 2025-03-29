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

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

type AdminHandler struct {
	bot         *tele.Bot
	userService *service.User
	chatService *service.Chat
	state       *State
	notify      *service.Notify
	log         *zap.Logger
}

func NewAdminHandler(
	bot *tele.Bot,
	userService *service.User,
	chatService *service.Chat,
	notificationService *service.Notify,
	state *State,
	log *zap.Logger,
) *AdminHandler {
	return &AdminHandler{
		bot:         bot,
		userService: userService,
		chatService: chatService,
		state:       state,
		notify:      notificationService,

		log: log,
	}
}

// StartAdmin handles the start command for admins
func (h *AdminHandler) StartAdmin(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.ManageUsers, menu.ManageChats),
		menu.AdminMenu.Row(menu.SendNotifyAdmin),
	)
	h.state.Set(c.Sender().ID, MenuState)

	return c.Send("Welcome, Admin! What would you like to do?", menu.AdminMenu)
}

func (h *AdminHandler) SendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, SendNotificationState)
	return c.Send("Please send me the message you want to send to all users.")
}

func (h *AdminHandler) ProcessSendNotification(c tele.Context) error {
	if h.state.Get(c.Sender().ID) != SendNotificationState {
		return nil
	}

	msg := c.Text()
	if msg == "" {
		return c.Send("Please send me the message you want to send to all users.")
	}

	// Создаем inline-клавиатуру с кнопками "Confirm" и "Cancel"
	confirmBtn := menu.Selector.Data("✅ Confirm", "confirm_notification", msg)

	menu.Selector.Inline(
		menu.Selector.Row(confirmBtn),
		menu.Selector.Row(menu.Selector.Data("❌ Cancel", "cancel_notification", msg)),
	)

	// Сохраняем состояние ожидания подтверждения
	h.state.Set(c.Sender().ID, ConfirmNotificationState)

	conf := "Are you sure you want to send this notification?\n\n"
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
	// Проверяем, что юзер в состоянии подтверждения
	if h.state.Get(c.Sender().ID) != ConfirmNotificationState {
		return nil
	}

	// Получаем сообщение из data кнопки
	msg := c.Data()

	// Отправляем всем пользователям
	num, err := h.notify.Broadcast(context.TODO(), h.bot, msg)
	if err != nil {
		return c.Send("Failed to send notification: " + err.Error())
	}

	h.state.Set(c.Sender().ID, MenuState)

	// Редактируем предыдущее сообщение, заменяя клавиатуру на текст
	return c.Edit("✅ Notification sent successfully to " + fmt.Sprintf("%d chats", num))
}

// Cancel sending notification
func (h *AdminHandler) CancelSendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, MenuState)
	return c.Edit("❌ Notification sending canceled.")
}

// ManageUsers handles the user management menu
func (h *AdminHandler) ManageUsers(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.AddUser, menu.RemoveUser),
		menu.AdminMenu.Row(menu.ListUser, menu.Back))
	h.state.Set(c.Sender().ID, MenuState)
	return c.Send("User Management. What would you like to do?", menu.AdminMenu)
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
	return c.Send("Please send me the Telegram username (@username) of the user you want to add.")
}

// ProcessAddUser processes the username input for adding a user
func (h *AdminHandler) ProcessAddUser(c tele.Context) error {
	userID := c.Sender().ID
	if h.state.Get(userID) != AddUserState {
		return nil
	}

	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send("Please send a valid username starting with @")
	}

	username = username[1:]

	if err := h.userService.Create(context.Background(), rand.Int64(), username, "", ""); err != nil {
		return c.Send("Failed to add user: " + err.Error())
	}

	h.state.Set(userID, MenuState) // Сбрасываем состояние
	return c.Send("User @" + username + " has been added.")
}

// RemoveUser handles removing a user
func (h *AdminHandler) RemoveUser(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveUserState)
	return c.Send(
		"Please send me the Telegram username (@username) of the user you want to remove.",
	)
}

// ProcessRemoveUser processes the username input for removing a user
func (h *AdminHandler) ProcessRemoveUser(c tele.Context) error {
	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send("Please send a valid username starting with @")
	}

	// Remove @ and extract the username
	username = username[1:]
	ctx := context.Background()

	// Call service to remove user
	err := h.userService.Delete(ctx, username)
	if err != nil {
		return c.Send("Failed to remove user: " + err.Error())
	}
	h.state.Set(c.Sender().ID, MenuState)

	return c.Send("User @" + username + " has been removed successfully!")
}

// ListUsers handles listing all users
func (h *AdminHandler) ListUsers(c tele.Context) error {
	users, err := h.userService.GetAll(context.TODO())
	if errors.Is(err, models.ErrNotFound) {
		return c.Send("No users found.")
	}
	if err != nil {
		return c.Send("Failed to get users: " + err.Error())
	}

	var response strings.Builder
	response.WriteString("📋 *User List:*\n\n")

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
	return c.Send("Chat Management. What would you like to do?", menu.AdminMenu)
}

// AddChat handles adding a new chat
func (h *AdminHandler) AddChat(c tele.Context) error {
	h.state.Set(c.Sender().ID, AddChatState)
	return c.Send("Please send me the chat username (@username) you want to add.")
}

// ProcessAddChat processes the chat input for adding a chat
func (h *AdminHandler) ProcessAddChat(c tele.Context) error {
	// Проверяем, откуда пришла команда (чат или личка)
	if c.Chat().Type == tele.ChatPrivate {
		return c.Send("This command must be used in a group or channel.")
	}

	chat := &models.Chat{
		ChatID:      c.Chat().ID,
		Title:       c.Chat().Title,
		Type:        string(c.Chat().Type),
		Description: c.Chat().Description,
	}

	err := h.chatService.Add(chat)
	if err != nil {
		return c.Send("Failed to add chat: " + err.Error())
	}

	return c.Send("Chat added successfully! The bot can now send notifications here.")
}

// RemoveChat handles removing a chat
func (h *AdminHandler) RemoveChat(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveChatState)
	return c.Send("Please send me the chat username (@username) you want to remove.")
}

// ProcessRemoveChat processes the chat input for removing a chat
func (h *AdminHandler) ProcessRemoveChat(c tele.Context) error {
	chatName := c.Text()
	if !strings.HasPrefix(chatName, "@") {
		return c.Send("Please send a valid chat username starting with @")
	}

	// Remove @ and extract the username
	chatName = chatName[1:]

	// Call service to remove chat
	err := h.chatService.Remove(chatName)
	if err != nil {
		return c.Send("Failed to remove chat: " + err.Error())
	}
	h.state.Set(c.Sender().ID, MenuState)
	return c.Send("Chat @" + chatName + " has been removed successfully!")
}

// ListChats handles listing all chats
func (h *AdminHandler) ListChats(c tele.Context) error {
	chats, err := h.chatService.GetAll()
	if err != nil {
		return c.Send("Failed to get chats: " + err.Error())
	}

	if len(chats) == 0 {
		return c.Send("No chats found.")
	}

	var response strings.Builder
	response.WriteString("📋 *Chat List:*\n\n")

	for i, chat := range chats {
		response.WriteString(
			fmt.Sprintf("%d. @%s\n", i+1, chat.Title),
		)
	}

	return c.Send(response.String(), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

// ShowStatistics shows bot statistics
func (h *AdminHandler) ShowStatistics(c tele.Context) error {
	return c.Send("Statistics feature is coming soon!")
}

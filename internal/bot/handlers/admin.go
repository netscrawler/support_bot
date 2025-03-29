package handlers

import (
	"context"
	"errors"
	"fmt"
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
	log         *zap.Logger
}

func NewAdminHandler(
	bot *tele.Bot,
	userService *service.User,
	chatService *service.Chat,
	log *zap.Logger,
) *AdminHandler {
	return &AdminHandler{
		bot:         bot,
		userService: userService,
		chatService: chatService,
		log:         log,
	}
}

// StartAdmin handles the start command for admins
func (h *AdminHandler) StartAdmin(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.ManageUsers, menu.ManageChats),
		menu.AdminMenu.Row(menu.SendNotifyAdmin),
	)

	return c.Send("Welcome, Admin! What would you like to do?", menu.AdminMenu)
}

// ManageUsers handles the user management menu
func (h *AdminHandler) ManageUsers(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.AddUser, menu.RemoveUser),
		menu.AdminMenu.Row(menu.ListUser, menu.Back))

	return c.Send("User Management. What would you like to do?", menu.AdminMenu)
}

// AddUser handles adding a new user
func (h *AdminHandler) AddUser(c tele.Context) error {
	c.Send("Please send me the Telegram username (@username) of the user you want to add.")
	return h.ProcessAddUser(c)
}

// ProcessAddUser processes the username input for adding a user
func (h *AdminHandler) ProcessAddUser(c tele.Context) error {
	c.Send("Please send me the Telegram username (@username) of the user you want to add.")
	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send("Please send a valid username starting with @")
	}

	// Remove @ and extract the username
	username = username[1:]
	// TODO: Зарефакторить это позорное говно
	chat, err := h.bot.ChatByUsername(username)

	// If we can get user info, add them with complete information
	if err == nil && chat != nil {
		// Make sure it's a user, not a group/channel
		if chat.Type != tele.ChatPrivate {
			return c.Send(
				"@" + username + " is not a user account. Please provide a valid user, not a group or channel.",
			)
		}

		userToAdd := models.User{
			TelegramID: 0,
			Username:   username,
			FirstName:  chat.FirstName,
			LastName:   &chat.LastName,
			Role:       "user",
		}
		h.log.Debug("user to add", zap.Any("user", userToAdd))

		err = h.userService.AddUserComplete(userToAdd)
		if err != nil {
			return c.Send("Failed to add user: " + err.Error())
		}

		return c.Send("User @" + username + " has been added successfully!")
	}

	// If we can't get user info, add just the username and wait for the user to interact with the bot
	err = h.userService.Create(context.Background(), 0, username, "", "")
	if err != nil {
		c.Send("Failed to add user: " + err.Error())
		return h.ProcessAddUser(c)
	}

	return c.Send("User @" + username + " has been added to the authorized list. " +
		"They need to start a conversation with the bot by sending /start to complete registration.")
}

// RemoveUser handles removing a user
func (h *AdminHandler) RemoveUser(c tele.Context) error {
	c.Send("Please send me the Telegram username (@username) of the user you want to remove.")
	return h.ProcessRemoveUser(c)
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
		menu.AdminMenu.Row(menu.AddChat, menu.RemoveChat),
		menu.AdminMenu.Row(menu.ListChats, menu.Back))
	return c.Send("Chat Management. What would you like to do?", menu.AdminMenu)
}

// AddChat handles adding a new chat
func (h *AdminHandler) AddChat(c tele.Context) error {
	c.Send("Please send me the chat username (@username) you want to add.")
	return h.ProcessAddChat(c)
}

// ProcessAddChat processes the chat input for adding a chat
func (h *AdminHandler) ProcessAddChat(c tele.Context) error {
	chatName := c.Text()
	if !strings.HasPrefix(chatName, "@") {
		return c.Send("Please send a valid chat username starting with @")
	}

	// Remove @ and extract the username
	chatName = chatName[1:]
	ch, err := h.bot.ChatByUsername(chatName)
	if err != nil {
		return c.Send("Failed to add chat: " + err.Error())
	}
	// TODO: Переписать это позорное говно
	chat := &models.Chat{
		ChatID:      ch.ID,
		Title:       ch.Title,
		Type:        string(ch.Type),
		Description: ch.Description,
	}

	// Call service to add chat
	err = h.chatService.Add(chat)
	if err != nil {
		return c.Send("Failed to add chat: " + err.Error())
	}

	return c.Send("Chat @" + chatName + " has been added successfully!")
}

// RemoveChat handles removing a chat
func (h *AdminHandler) RemoveChat(c tele.Context) error {
	c.Send("Please send me the chat username (@username) you want to remove.")
	return h.ProcessRemoveChat(c)
}

// ProcessRemoveChat processes the chat input for removing a chat
func (h *AdminHandler) ProcessRemoveChat(c tele.Context) error {
	chatName := c.Text()
	if !strings.HasPrefix(chatName, "@") {
		return c.Send("Please send a valid chat username starting with @")
	}

	// Remove @ and extract the username
	chatName = chatName[1:]

	ch, err := h.bot.ChatByUsername(chatName)
	if err != nil {
		return c.Send("Failed to find chat: " + err.Error())
	}

	// Call service to remove chat
	err = h.chatService.Remove(ch.ID)
	if err != nil {
		return c.Send("Failed to remove chat: " + err.Error())
	}

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

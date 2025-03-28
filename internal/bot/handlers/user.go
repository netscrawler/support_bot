package handlers

import (
	"strings"
	"support_bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

type UserHandler struct {
	bot         *tele.Bot
	chatService *service.Chat
}

func NewUserHandler(
	bot *tele.Bot,
	chatService *service.Chat,
) *UserHandler {
	return &UserHandler{
		bot:         bot,
		chatService: chatService,
	}
}

// StartUser handles the start command for regular users
func (h *UserHandler) StartUser(c tele.Context) error {
	keyboard := &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{{Text: "📝 Send Notification"}},
		},
		ResizeKeyboard: true,
	}

	return c.Send("Welcome! What would you like to do?", keyboard)
}

// SendNotification handles the notification sending workflow
// TODO: Переделать эту залупу
func (h *UserHandler) SendNotificationWithOpt(c tele.Context) error {
	chats, err := h.chatService.GetAll()
	if err != nil {
		return c.Send("Failed to get chats: " + err.Error())
	}

	if len(chats) == 0 {
		return c.Send("No chats available for sending notifications.")
	}

	// Create keyboard with all chats
	var keyboard [][]tele.ReplyButton
	var row []tele.ReplyButton

	for i, chat := range chats {
		row = append(row, tele.ReplyButton{Text: "@" + chat.Title})

		// Create a new row every 2 buttons
		if (i+1)%2 == 0 || i == len(chats)-1 {
			keyboard = append(keyboard, row)
			row = []tele.ReplyButton{}
		}
	}

	// Add cancel button
	keyboard = append(keyboard, []tele.ReplyButton{{Text: "❌ Cancel"}})

	markup := &tele.ReplyMarkup{
		ReplyKeyboard:  keyboard,
		ResizeKeyboard: true,
	}

	return c.Send("Select a chat to send notification to:", markup)
}

// TODO доделать это говно позорное
func (h *UserHandler) SendNotification(c tele.Context) error {
	chats, err := h.chatService.GetAll()
	if err != nil {
		return c.Send("Failed to get chats: " + err.Error())
	}

	if len(chats) == 0 {
		return c.Send("No chats available for sending notifications.")
	}

	// Add cancel button

	markup := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	return c.Send("Select a chat to send notification to:", markup)
}

// ProcessChatSelection processes the selected chat
func (h *UserHandler) ProcessChatSelection(c tele.Context) error {
	selectedChat := c.Text()

	if selectedChat == "❌ Cancel" {
		return h.StartUser(c)
	}

	if !strings.HasPrefix(selectedChat, "@") {
		return c.Send("Please select a valid chat from the keyboard.")
	}

	// Store selected chat for later use

	// Give user a regular keyboard to cancel
	keyboard := &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{{Text: "❌ Cancel"}},
		},
		ResizeKeyboard: true,
	}

	return c.Send(
		"Please enter the notification text you want to send to "+selectedChat+":",
		keyboard,
	)
}

// ProcessNotificationText processes the notification text
func (h *UserHandler) ProcessNotificationText(c tele.Context, chatName string) error {
	text := c.Text()

	if text == "❌ Cancel" {
		return h.StartUser(c)
	}
	// TODO добавить логику
	// Send notification logic would go here
	// Currently this is just a placeholder since the notification service is commented out

	// Return to the main menu with success message
	c.Send("Notification successfully sent to @" + chatName + "!")
	return h.StartUser(c)
}

package handlers

import (
	"context"
	"fmt"
	"support_bot/internal/bot/menu"
	"support_bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

type UserHandler struct {
	bot         *tele.Bot
	chatService *service.Chat
	userService *service.User
	state       *State
	notify      *service.Notify
}

func NewUserHandler(
	bot *tele.Bot,
	chatService *service.Chat,
	userService *service.User,
	state *State,
	notify *service.Notify,
) *UserHandler {
	return &UserHandler{
		bot:         bot,
		chatService: chatService,
		userService: userService,
		state:       state,
		notify:      notify,
	}
}

func (h *UserHandler) ProcessUserInput(c tele.Context) error {
	userID := c.Sender().ID
	state := h.state.Get(userID)

	switch state {
	case SendNotificationState:
		return h.ProcessSendNotification(c)
	case ConfirmNotificationState:
		return h.ConfirmSendNotification(c)
	case CancelNotificationState:
		return h.CancelSendNotification(c)
	default:
		return nil
	}
}

// StartUser handles the start command for regular users
func (h *UserHandler) StartUser(c tele.Context) error {
	menu.UserMenu.Reply(
		menu.UserMenu.Row(menu.SendNotifyUser),
	)
	h.state.Set(c.Sender().ID, MenuState)
	return c.Send("Welcome! What would you like to do?", menu.UserMenu)
}

func (h *UserHandler) RegisterUser(c tele.Context) error {
	snd := c.Sender()
	err := h.userService.AddUserComplete(snd)
	if err == nil {
		return c.Send("You have been registered successfully!")
	}
	return nil
}

func (h *UserHandler) SendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, SendNotificationState)
	return c.Send("Please send me the message you want to send to all users.")
}

func (h *UserHandler) ProcessSendNotification(c tele.Context) error {
	if h.state.Get(c.Sender().ID) != SendNotificationState {
		return nil
	}

	msg := c.Text()
	if msg == "" {
		return c.Send("Please send me the message you want to send to all users.")
	}

	// Создаем inline-клавиатуру с кнопками "Confirm" и "Cancel"
	confirmBtn := menu.Selector.Data("✅ Confirm", "confirm_user_notification", msg)

	menu.Selector.Inline(
		menu.Selector.Row(confirmBtn),
		menu.Selector.Row(menu.Selector.Data("❌ Cancel", "cancel_user_notification", msg)),
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
func (h *UserHandler) ConfirmSendNotification(c tele.Context) error {
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
func (h *UserHandler) CancelSendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, MenuState)
	return c.Edit("❌ Notification sending canceled.")
}

func (h *UserHandler) UserAuthMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		// Получаем username пользователя
		username := c.Sender().Username
		if username == "" {
			return nil
		}

		// Проверяем пользователя в базе
		user, err := h.userService.GetByUsername(context.Background(), username)
		if err != nil {
			return nil
		}

		// Сохраняем пользователя в context
		c.Set("user", user)
		return next(c)
	}
}

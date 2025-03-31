package handlers

import (
	"context"
	"errors"
	"fmt"
	"support_bot/internal/bot/menu"
	"support_bot/internal/models"
	"support_bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

type UserHandler struct {
	bot         *tele.Bot
	chatService *service.Chat
	userService *service.User
	state       *State
	chatNotify  *service.ChatNotify
	userNotify  *service.UserNotify
}

func NewUserHandler(
	bot *tele.Bot,
	chatService *service.Chat,
	userService *service.User,
	state *State,
	chatNotify *service.ChatNotify,
	userNotify *service.UserNotify,
) *UserHandler {
	return &UserHandler{
		bot:         bot,
		chatService: chatService,
		userService: userService,
		state:       state,
		chatNotify:  chatNotify,
		userNotify:  userNotify,
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

func (h *UserHandler) StartUser(c tele.Context) error {
	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}

	menu.UserMenu.Reply(
		menu.UserMenu.Row(menu.SendNotifyUser),
	)
	//nolint:errcheck
	c.Delete()
	h.state.Set(c.Sender().ID, MenuState)
	return c.Send("Добро пожаловать!", menu.UserMenu)
}

func (h *UserHandler) RegisterUser(c tele.Context) error {
	//nolint:errcheck
	c.Delete()
	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}
	ctx := context.Background()
	snd := c.Sender()
	err := h.userService.AddUserComplete(snd)
	formatedString := fmt.Sprintf(
		"Пользователь с ником @%s успешно прошел регистрацию",
		c.Sender().Username,
	)
	// nolint:errcheck
	h.userNotify.SendAdminNotify(ctx, h.bot, formatedString)
	if err == nil {
		return c.Send("Вы успешно прошли регистрацию!\n напишите /start чтобы начать работу")
	}
	return nil
}

func (h *UserHandler) SendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, SendNotificationState)
	//nolint:errcheck
	c.Delete()
	return c.Send("Пожалуйста, пришлите мне сообщение, которое вы хотите отправить.")
}

func (h *UserHandler) ProcessSendNotification(c tele.Context) error {
	if h.state.Get(c.Sender().ID) != SendNotificationState {
		return c.Edit("Время на отправку истекло, начните заново")
	}

	msg := c.Text()
	if msg == "" {
		return c.Send("Пожалуйста, пришлите мне сообщение, которое вы хотите отправить.")
	}

	h.state.SetMsgData(c.Sender().ID, msg)
	confirmBtn := menu.Selector.Data(
		"✅ Отправить",
		"confirm_user_notification",
	)
	cancelBtn := menu.Selector.Data("❌ Отменить", "cancel_user_notification")

	menu.Selector.Inline(
		menu.Selector.Row(cancelBtn, confirmBtn),
	)

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

func (h *UserHandler) ConfirmSendNotification(c tele.Context) error {
	ctx := context.Background()
	msg, ok := h.state.GetMsgData(c.Sender().ID)
	if h.state.Get(c.Sender().ID) != ConfirmNotificationState || !ok {
		return c.Edit("Время на подтверждение истекло")
	}

	fmt.Println("start broadcast")
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
	go h.userNotify.SendAdminNotify(ctx, h.bot, formString)

	h.state.Set(c.Sender().ID, MenuState)
	return c.Edit(resp, tele.ModeMarkdownV2)
}

func (h *UserHandler) CancelSendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, MenuState)
	return c.Edit("❌ Отправка уведомления отменена.")
}

func (h *UserHandler) UserAuthMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		// Получаем username пользователя
		username := c.Sender().Username
		if username == "" {
			return nil
		}

		// Проверяем пользователя в базе
		user, err := h.userService.GetByUsername(ctx, username)
		if err != nil {
			return nil
		}

		// Сохраняем пользователя в context
		c.Set("user", user)
		return next(c)
	}
}
